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

When using HTTP transport the server listens on `http://HOST:PORT/mcp` (the
`/mcp` path is fixed).  The default listen address is `0.0.0.0:8483`; override
it with `-listen HOST:PORT`.

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

## Integration approaches

There are two ways to connect an AI agent to the MCP server.

### stdio (agent-managed)

The AI client starts and stops `slackdump mcp` automatically.  The archive
path is baked into the client config, so switching archives requires editing
that config and restarting the client.

### HTTP (terminal-managed) — recommended for interactive use

You start the server yourself in a terminal, point it at any archive, and the
AI client connects over HTTP.  To switch archives, stop the server, run it
again with a different path — no reconfiguration of the AI client needed.

**Step 1** — start the server in a terminal:

```
slackdump mcp -transport http /path/to/your/archive.db
```

The server prints the endpoint address and listens on
`http://localhost:8483/mcp`.

**Step 2** — in another terminal (or tab), start your AI agent as usual
(`opencode`, `claude`, etc.).  Because the MCP connection is already
established, you can switch archives at any time by restarting only the
slackdump process, without touching the agent session.

To use a non-default port:

```
slackdump mcp -transport http -listen 127.0.0.1:9000 /path/to/your/archive.db
```

Note: a plain `GET http://localhost:8483/mcp` hangs by design — that endpoint
streams server-sent events (SSE).  Only `POST` requests carry MCP messages.

## Integrating with Claude Desktop

Claude Desktop manages the process itself (stdio only).  Add to your
`claude_desktop_config.json`:

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

### HTTP (recommended — switch archives without leaving OpenCode)

Start the server in one terminal:

```
slackdump mcp -transport http /path/to/your/archive.db
```

Add to your `~/.config/opencode/config.json`:

```json
{
  "mcp": {
    "slackdump": {
      "type": "remote",
      "url": "http://localhost:8483/mcp"
    }
  }
}
```

To point at a different archive, restart only the slackdump process.
OpenCode reconnects automatically on the next tool call.

### stdio (OpenCode manages the process)

Add to your `~/.config/opencode/config.json`:

```json
{
  "mcp": {
    "slackdump": {
      "type": "local",
      "command": ["slackdump", "mcp", "/path/to/your/archive.db"]
    }
  }
}
```

To switch archives you must update the config and restart OpenCode.

## Flags
