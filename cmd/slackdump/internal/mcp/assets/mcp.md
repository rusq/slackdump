# slackdump mcp

Start a local **Model Context Protocol (MCP)** server that exposes Slackdump
archive data to AI agents (such as GitHub Copilot, Claude Desktop, or any MCP
client).

The server is read-only: it never modifies the underlying archive.

## Usage

```
slackdump mcp [flags] <archive>
```

`<archive>` is a path to any Slackdump archive: a SQLite database (`.db` or
`.sqlite`), a chunk directory, a Slack Export ZIP / directory, or a Dump ZIP /
directory.  The format is auto-detected.

## Transport

By default the server communicates over **stdio**, which is the standard
transport for local MCP integrations.  Pass `-transport http` to start an HTTP
server instead (useful for remote agents or multiple simultaneous clients).

## Available MCP Tools

| Tool | Description |
|------|-------------|
| `list_channels` | List all channels in the archive |
| `get_channel` | Get detailed info for a channel by ID |
| `list_users` | List all users/members |
| `get_messages` | Read messages from a channel (paginated) |
| `get_thread` | Read all replies in a thread |
| `get_workspace_info` | Workspace / team metadata |
| `command_help` | Get CLI flag help for any slackdump subcommand |

## Integrating with Claude Desktop

Add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "slackdump": {
      "command": "slackdump",
      "args": ["mcp", "/path/to/your/archive.db"]
    }
  }
}
```

## Integrating with VS Code (GitHub Copilot)

Add to your workspace `.vscode/mcp.json`:

```json
{
  "servers": {
    "slackdump": {
      "type": "stdio",
      "command": "slackdump",
      "args": ["mcp", "/path/to/your/archive.db"]
    }
  }
}
```

## Integrating with OpenCode

Add to your `~/.config/opencode/config.json` (or `~/.opencode/config.json`):

```json
{
  "mcp": {
    "servers": {
      "slackdump": {
        "type": "local",
        "command": "slackdump",
        "args": ["mcp", "/path/to/your/archive.db"]
      }
    }
  }
}
```

## Flags
