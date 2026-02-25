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

// Package mcp contains the CLI command for starting the Slackdump MCP server.
package mcp

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	internalmcp "github.com/rusq/slackdump/v4/internal/mcp"
	"github.com/rusq/slackdump/v4/source"
)

//go:embed assets/mcp.md
var mdMCP string

// CmdMCP is the "slackdump mcp" command.
var CmdMCP = &base.Command{
	UsageLine:   "slackdump mcp [flags] [<archive>]",
	Short:       "Start a local MCP server for an archive",
	Long:        mdMCP,
	FlagMask:    cfg.OmitAll,
	PrintFlags:  true,
	RequireAuth: false,
	Run:         runMCP,
}

var (
	listenAddr string
	transport  string
)

func init() {
	CmdMCP.Flag.StringVar(&transport, "transport", "stdio", "MCP transport: \"stdio\" or \"http\"")
	CmdMCP.Flag.StringVar(&listenAddr, "listen", "127.0.0.1:8483", "address to listen on when -transport=http")
}

func runMCP(ctx context.Context, cmd *base.Command, args []string) error {
	lg := cfg.Log

	var mcpOpts []internalmcp.Option
	mcpOpts = append(mcpOpts, internalmcp.WithLogger(lg))

	if len(args) >= 1 {
		archivePath := args[0]
		lg.InfoContext(ctx, "mcp: opening archive", "path", archivePath)

		src, err := source.Load(ctx, archivePath)
		if err != nil {
			base.SetExitStatus(base.SUserError)
			return fmt.Errorf("mcp: open archive: %w", err)
		}
		defer src.Close()

		mcpOpts = append(mcpOpts, internalmcp.WithSource(src))
	} else {
		lg.InfoContext(ctx, "mcp: no archive specified; agent must call load_source before using data tools")
	}

	srv := internalmcp.New(mcpOpts...)

	// Add the command_help tool at the CLI layer because it needs access to
	// cmd/slackdump/internal packages which are forbidden from internal/mcp.
	srv.AddTool(toolCommandHelp())

	switch strings.ToLower(transport) {
	case "stdio", "":
		return srv.ServeStdio(ctx)
	case "http":
		lg.InfoContext(ctx, "mcp: http transport", "addr", listenAddr)
		return srv.ServeHTTP(ctx, listenAddr)
	default:
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("mcp: unknown transport %q (use \"stdio\" or \"http\")", transport)
	}
}

// ─── command_help tool ────────────────────────────────────────────────────────

// toolCommandHelp returns an MCP tool that provides CLI flag help for any
// slackdump subcommand.  It lives at the CLI layer so it can access
// cmd/slackdump/internal packages.
func toolCommandHelp() mcpsrv.ServerTool {
	tool := mcplib.NewTool("command_help",
		mcplib.WithDescription(`Return command-line flag help for a slackdump subcommand.

Providing no command name (or an empty string) returns the top-level help
listing all available commands. This is useful when you need to construct a
slackdump command invocation and want to know what flags are available.`),
		mcplib.WithString("command",
			mcplib.Description(`Subcommand name, e.g. "archive", "export", "dump", "view". Leave empty for top-level help. Nested subcommands can be space-separated, e.g. "workspace new".`),
		),
		mcplib.WithReadOnlyHintAnnotation(true),
		mcplib.WithIdempotentHintAnnotation(true),
	)
	return mcpsrv.ServerTool{Tool: tool, Handler: handleCommandHelp}
}

func handleCommandHelp(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	args := req.GetArguments()
	cmdName := ""
	if args != nil {
		if v, ok := args["command"]; ok {
			cmdName, _ = v.(string)
		}
	}

	var buf bytes.Buffer

	if cmdName == "" {
		fmt.Fprintln(&buf, "Slackdump — available commands:")
		for _, c := range base.Slackdump.Commands {
			if c.Short == "" {
				continue
			}
			fmt.Fprintf(&buf, "  %-20s %s\n", c.Name(), c.Short)
		}
		return mcplib.NewToolResultText(buf.String()), nil
	}

	// Walk the command tree using the supplied name parts.
	parts := strings.Fields(cmdName)
	cur := base.Slackdump
	for _, part := range parts {
		found := false
		for _, sub := range cur.Commands {
			if sub.Name() == part {
				cur = sub
				found = true
				break
			}
		}
		if !found {
			return mcplib.NewToolResultText(fmt.Sprintf(
				"Unknown command %q. Run command_help with an empty command name to list all commands.",
				cmdName,
			)), nil
		}
	}

	fmt.Fprintf(&buf, "Command: slackdump %s\n", cur.LongName())
	if cur.Short != "" {
		fmt.Fprintf(&buf, "Summary: %s\n", cur.Short)
	}
	if cur.Long != "" {
		fmt.Fprintf(&buf, "\nDescription:\n%s\n", cur.Long)
	}

	if cur.PrintFlags || cur.FlagMask != cfg.OmitAll {
		fmt.Fprintln(&buf, "\nFlags:")
		if !cur.CustomFlags {
			cfg.SetBaseFlags(&cur.Flag, cur.FlagMask)
		}
		cur.Flag.SetOutput(&buf)
		cur.Flag.PrintDefaults()
	}

	if len(cur.Commands) > 0 {
		fmt.Fprintln(&buf, "\nSubcommands:")
		for _, sub := range cur.Commands {
			if sub.Short != "" {
				fmt.Fprintf(&buf, "  %-20s %s\n", sub.Name(), sub.Short)
			}
		}
	}

	return mcplib.NewToolResultText(buf.String()), nil
}
