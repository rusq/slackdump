// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package mcp

// In this file: MCP server construction and transport management.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
	"golang.org/x/sync/semaphore"

	"github.com/rusq/slackdump/v4/source"
)

const (
	serverName    = "slackdump-mcp"
	serverVersion = "1.0.0"

	// Readers acquire one token; replacement and Close acquire all tokens.
	sourceAccessCapacity int64 = 1 << 62
)

// Transport selects how the MCP server communicates with its client.
type Transport string

const (
	// TransportStdio uses stdin/stdout for communication (default, suitable
	// for local agent integrations such as Claude Desktop).
	TransportStdio Transport = "stdio"
	// TransportHTTP uses Streamable HTTP transport (suitable for remote
	// agents or when multiple concurrent clients are needed).
	TransportHTTP Transport = "http"
)

// SourceLoader is a function that opens a source by path and returns a
// SourceResumeCloser.  It is used by the load_source tool to open new sources
// at runtime.  [source.Load] is the standard implementation.
type SourceLoader func(ctx context.Context, path string) (source.SourceResumeCloser, error)

var errServerClosed = errors.New("mcp server is closed")

// Server wraps an MCP server and its underlying data source.
type Server struct {
	mcp    *mcpsrv.MCPServer
	logger *slog.Logger

	// access protects the entire lifetime of source reads, including iteration.
	// Weighted acquisition gives writers exclusive access without making waits
	// uncancellable. These fields are protected by access after construction.
	access *semaphore.Weighted
	src    source.SourceResumeCloser // nil until a source is loaded
	closed bool
	loader SourceLoader
}

// Option is a functional option for [New].
type Option func(*Server)

// WithSource transfers ownership of src to the server, which closes it on
// replacement or Close. It pre-loads a source so that the server is ready to
// answer tool calls.  If this option is not provided the server starts without
// a source and the agent must call the load_source tool before any data tool
// will work.
func WithSource(src source.SourceResumeCloser) Option {
	return func(s *Server) {
		s.src = src
	}
}

// WithLogger sets the logger used by the server.  A nil value is silently
// ignored (the default logger is used in that case).
func WithLogger(lg *slog.Logger) Option {
	return func(s *Server) {
		if lg != nil {
			s.logger = lg
		}
	}
}

// WithSourceLoader overrides the function used by the load_source tool to open
// archive files.  The default is [source.Load].  This option is primarily
// useful for testing.
func WithSourceLoader(fn SourceLoader) Option {
	return func(s *Server) {
		if fn != nil {
			s.loader = fn
		}
	}
}

// New creates a new MCP server.  The server is populated with all available
// tools but does not start listening until one of the Serve* methods is called.
//
// Use [WithSource] to pre-load an archive, [WithLogger] to set a custom
// logger, and [WithSourceLoader] to override the source-open function used by
// the load_source tool.
func New(opts ...Option) *Server {
	s := &Server{
		logger: slog.Default(),
		loader: source.Load,
		access: semaphore.NewWeighted(sourceAccessCapacity),
	}
	for _, o := range opts {
		o(s)
	}

	mcpServer := mcpsrv.NewMCPServer(
		serverName,
		serverVersion,
		mcpsrv.WithInstructions(instructions()),
	)

	// Register all tools.
	for _, t := range s.tools() {
		mcpServer.AddTool(t.Tool, t.Handler)
	}

	s.mcp = mcpServer
	return s
}

// instructions describes the active-source model without embedding startup
// state that would become stale after load_source.
func instructions() string {
	return `You are connected to a Slackdump MCP server.

Data tools read the currently loaded archive. If no archive is loaded, use
load_source first. Use load_source to switch archives at any time; its response
identifies the archive loaded by that call.

Available tools allow you to:
- List all channels in the archive (list_channels)
- Look up a channel (get_channel)
- List all users/members (list_users)
- Read messages from a channel, paginated (get_messages)
- Read thread replies (get_thread)
- Get workspace information (get_workspace_info)
- Get command-line flag help for slackdump subcommands (command_help)

All data is read-only. Timestamps in messages use Slack's format (Unix epoch as decimal string, e.g. "1609459200.000001").
`
}

// withSource holds shared access until fn returns, including iterator cleanup
// and response serialization. The source and its iterators must not escape fn.
// fn must not acquire access again: a pending writer blocks recursive reads.
func (s *Server) withSource(ctx context.Context, fn func(source.Sourcer) (*mcplib.CallToolResult, error)) (*mcplib.CallToolResult, error) {
	if err := s.access.Acquire(ctx, 1); err != nil {
		return resultErr(err), nil
	}
	defer s.access.Release(1)
	if s.closed {
		return resultErr(errServerClosed), nil
	}
	if s.src == nil {
		return resultErr(errNoSource), nil
	}
	return fn(s.src)
}

// loadSource waits for active readers, closes the current source (if any), and
// replaces it with next. On cancellation or shutdown, next remains caller-owned.
func (s *Server) loadSource(ctx context.Context, next source.SourceResumeCloser) error {
	if err := s.access.Acquire(ctx, sourceAccessCapacity); err != nil {
		return err
	}
	defer s.access.Release(sourceAccessCapacity)
	if s.closed {
		return errServerClosed
	}
	if s.src != nil {
		if err := s.src.Close(); err != nil {
			// Log but do not abort — we still want to switch sources.
			s.logger.Warn("mcp: error closing previous source", "err", err)
		}
	}
	s.src = next
	return nil
}

// Close waits for active source operations and closes the current source.
// It is idempotent and permanently rejects subsequent data and load operations.
// Callers should stop serving before calling Close. An in-flight load that has
// not published its source will be rejected and must close its own source.
func (s *Server) Close() error {
	// Shutdown must finish cleanup even if the serving context was canceled.
	_ = s.access.Acquire(context.Background(), sourceAccessCapacity)
	defer s.access.Release(sourceAccessCapacity)
	s.closed = true
	src := s.src
	s.src = nil
	if src != nil {
		return src.Close()
	}
	return nil
}

// ServeStdio runs the MCP server over stdin/stdout until ctx is cancelled.
// This is the standard transport used by local agent integrations.
func (s *Server) ServeStdio(ctx context.Context) error {
	srv := mcpsrv.NewStdioServer(s.mcp)
	s.logger.InfoContext(ctx, "mcp server listening on stdio")
	if err := srv.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("mcp stdio server error: %w", err)
	}
	return nil
}

// ServeHTTP runs the MCP server as a Streamable HTTP server on addr until
// ctx is cancelled.  addr should be a host:port string such as "127.0.0.1:8483".
// The MCP endpoint is available at /mcp on that address.
func (s *Server) ServeHTTP(ctx context.Context, addr string) error {
	// Do NOT pass WithStreamableHTTPServer — when a pre-built *http.Server is
	// provided, Start() skips creating the ServeMux that registers /mcp, so
	// every request returns 404.  Let Start() build its own mux instead.
	streamSrv := mcpsrv.NewStreamableHTTPServer(s.mcp)

	s.logger.InfoContext(ctx, "mcp server listening on http", "addr", addr, "endpoint", addr+"/mcp")

	errCh := make(chan error, 1)
	go func() {
		if err := streamSrv.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("mcp http server error: %w", err)
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		s.logger.InfoContext(ctx, "mcp server shutting down")
		if err := streamSrv.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("mcp http server shutdown error: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

// tools returns all MCP tools that this server exposes.
func (s *Server) tools() []mcpsrv.ServerTool {
	return []mcpsrv.ServerTool{
		s.toolLoadSource(),
		s.toolListChannels(),
		s.toolGetChannel(),
		s.toolListUsers(),
		s.toolGetMessages(),
		s.toolGetThread(),
		s.toolGetWorkspaceInfo(),
	}
}

// AddTool adds an additional tool to the MCP server.  This can be called after
// New but before serving starts.  It is intended for CLI-layer tools that have
// access to internal CLI packages (e.g. command_help).
func (s *Server) AddTool(tool mcpsrv.ServerTool) {
	s.mcp.AddTool(tool.Tool, tool.Handler)
}

// resultText is a helper that wraps text in a successful CallToolResult.
func resultText(text string) *mcplib.CallToolResult {
	return mcplib.NewToolResultText(text)
}

// resultErr is a helper that wraps an error in a CallToolResult with IsError=true.
func resultErr(err error) *mcplib.CallToolResult {
	return &mcplib.CallToolResult{
		Content: []mcplib.Content{mcplib.NewTextContent(err.Error())},
		IsError: true,
	}
}

// resultJSON serialises v to JSON and returns a CallToolResult with the JSON
// as plain text content.  We intentionally do not populate StructuredContent
// because the MCP spec requires it to be a JSON object (record), which is
// violated when v is a slice — causing clients to reject the response.
func resultJSON(v any) (*mcplib.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal JSON: %w", err)
	}
	return mcplib.NewToolResultText(string(b)), nil
}

// stringArg extracts a named string argument from a tool call request.
// Returns ("", false) if the argument is absent or not a string.
func stringArg(req mcplib.CallToolRequest, name string) (string, bool) {
	args := req.GetArguments()
	if args == nil {
		return "", false
	}
	v, ok := args[name]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// intArg extracts a named int argument from a tool call request.  The MCP
// protocol serialises numbers as float64, so we convert accordingly.
func intArg(req mcplib.CallToolRequest, name string, defaultVal int) int {
	args := req.GetArguments()
	if args == nil {
		return defaultVal
	}
	v, ok := args[name]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return defaultVal
}

// boolArg extracts a named bool argument from a tool call request.
func boolArg(req mcplib.CallToolRequest, name string, defaultVal bool) bool {
	args := req.GetArguments()
	if args == nil {
		return defaultVal
	}
	v, ok := args[name]
	if !ok {
		return defaultVal
	}
	b, ok := v.(bool)
	if !ok {
		return defaultVal
	}
	return b
}
