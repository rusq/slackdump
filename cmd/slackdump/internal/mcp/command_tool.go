package mcp

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
)

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
