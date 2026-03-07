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
	"sync"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"

	"github.com/rusq/slackdump/v4/source"
)

const (
	serverName    = "slackdump-mcp"
	serverVersion = "1.0.0"
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

// Server wraps an MCP server and its underlying data source.
type Server struct {
	mcp    *mcpsrv.MCPServer
	logger *slog.Logger

	// mu protects src.  All tool handlers must hold mu.RLock while reading src
	// and load_source must hold mu.Lock while replacing it.
	mu     sync.RWMutex
	src    source.SourceResumeCloser // nil until a source is loaded
	loader SourceLoader              // used by load_source to open new sources
}

// Option is a functional option for [New].
type Option func(*Server)

// WithSource pre-loads a source so that the server is immediately ready to
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
	}
	for _, o := range opts {
		o(s)
	}

	mcpServer := mcpsrv.NewMCPServer(
		serverName,
		serverVersion,
		mcpsrv.WithInstructions(instructions(s.src)),
	)

	// Register all tools.
	for _, t := range s.tools() {
		mcpServer.AddTool(t.Tool, t.Handler)
	}

	s.mcp = mcpServer
	return s
}

// instructions returns the server instructions that describe the data source
// to the connecting agent.  When src is nil (no source loaded yet) a generic
// prompt is returned that asks the agent to call load_source first.
func instructions(src source.Sourcer) string {
	if src == nil {
		return `You are connected to a Slackdump MCP server.

No archive has been loaded yet. Use the load_source tool to open a Slackdump
archive before calling any other data tools.

Once a source is loaded the following tools become available:
- list_channels  – list all channels in the archive
- list_users     – list all users/members
- get_messages   – read messages from a channel (paginated)
- get_thread     – read thread replies
- get_workspace_info – get workspace information
`
	}
	return fmt.Sprintf(`You are connected to a Slackdump MCP server.

The archive "%s" (type: %s) contains exported Slack workspace data.

Available tools allow you to:
- List all channels in the archive
- List all users/members
- Read messages from a channel (paginated)
- Read thread replies
- Get workspace information
- Get command-line flag help for slackdump subcommands

All data is read-only. Timestamps in messages use Slack's format (Unix epoch as decimal string, e.g. "1609459200.000001").
`, src.Name(), src.Type())
}

// source returns the current source under a read-lock.  Returns nil when no
// source has been loaded.
func (s *Server) source() source.SourceResumeCloser {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.src
}

// loadSource closes the current source (if any) and replaces it with next.
// It holds the write-lock for the duration of the swap.
func (s *Server) loadSource(next source.SourceResumeCloser) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.src != nil {
		if err := s.src.Close(); err != nil {
			// Log but do not abort — we still want to switch sources.
			s.logger.Warn("mcp: error closing previous source", "err", err)
		}
	}
	s.src = next
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
