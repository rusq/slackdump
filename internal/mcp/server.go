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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

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

// Server wraps an MCP server and its underlying data source.
type Server struct {
	mcp    *mcpsrv.MCPServer
	src    source.Sourcer
	logger *slog.Logger
}

// New creates a new MCP server backed by the given Sourcer.  The server is
// populated with all available tools but does not start listening until one of
// the Serve* methods is called.
func New(src source.Sourcer, lg *slog.Logger) *Server {
	if lg == nil {
		lg = slog.Default()
	}
	s := &Server{
		src:    src,
		logger: lg,
	}

	mcpServer := mcpsrv.NewMCPServer(
		serverName,
		serverVersion,
		mcpsrv.WithInstructions(instructions(src)),
	)

	// Register all tools.
	for _, t := range s.tools() {
		mcpServer.AddTool(t.Tool, t.Handler)
	}

	s.mcp = mcpServer
	return s
}

// instructions returns the server instructions that describe the data source
// to the connecting agent.
func instructions(src source.Sourcer) string {
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
func (s *Server) ServeHTTP(ctx context.Context, addr string) error {
	httpSrv := &http.Server{Addr: addr}
	streamSrv := mcpsrv.NewStreamableHTTPServer(s.mcp,
		mcpsrv.WithStreamableHTTPServer(httpSrv),
	)

	s.logger.InfoContext(ctx, "mcp server listening on http", "addr", addr)

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

// resultJSON is a helper that serialises v to JSON and returns a CallToolResult.
func resultJSON(v any) (*mcplib.CallToolResult, error) {
	return mcplib.NewToolResultJSON(v)
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
