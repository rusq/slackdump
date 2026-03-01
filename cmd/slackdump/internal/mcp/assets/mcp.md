# slackdump mcp

Start a local **Model Context Protocol (MCP)** server that exposes Slackdump
archive data to AI agents (such as GitHub Copilot, Claude Desktop, or any MCP
client).

The server is read-only: it never modifies the underlying archive.

## Usage

### Running MCP
```
slackdump mcp [flags] [archive]
```

`archive` is a path to any Slackdump archive: a SQLite database (`.db` or
`.sqlite`), a chunk directory, a Slack Export ZIP / directory, or a Dump ZIP /
directory.  The format is auto-detected.

The archive path is optional.  If omitted, the server starts without a loaded
source and the agent must call the `load_source` tool before any data tool will
work.  This is useful when the agent itself decides which archive to open, or
when you want to switch archives at runtime without restarting the server.

### Creating a new project

Scaffold a ready-to-use AI project directory pre-configured for a specific AI
tool.  The directory is created if it does not exist.

```
slackdump mcp -new <layout> <directory>
```

`<layout>` is the project layout to create.  Currently supported:

- **`opencode`** — creates `opencode.jsonc` wiring up the MCP server, plus
  three OpenCode skills (`slackdump`, `slackdump-source`, `slackdump-sqlite3`)
  inside `.opencode/skills/`.

- **`claude-code`** — creates `.mcp.json` (project-scoped MCP config for
  Claude Code), plus skill content in `CLAUDE.md` (main guidance) and
  `.claude/slackdump-source.md` / `.claude/slackdump-sqlite3.md`.

- **`copilot`** — creates `.vscode/mcp.json` wiring up the MCP server for VS
  Code / GitHub Copilot, plus `.github/copilot-instructions.md` (always-on
  guidance) and two file-scoped instruction files in `.github/instructions/`.

**Example — set up an OpenCode project:**

```
slackdump mcp -new opencode ~/my-slack-project
```

After running this command:

1. `~/my-slack-project/opencode.jsonc` configures the Slackdump MCP server as
   a local stdio process — OpenCode will start and stop it automatically.
2. `~/my-slack-project/.opencode/skills/` contains three skills that teach
   OpenCode how to work with Slackdump archives, fall back to direct SQLite
   access, and understand the different source formats.

Open the project directory in OpenCode to start working:

```
opencode ~/my-slack-project
```

The agent will have the Slackdump MCP tools available immediately and will
use the bundled skills for guidance.  Call `load_source` (or pass an archive
path when starting `slackdump mcp`) to point the server at an archive.

**Example — set up a Claude Code project:**

```
slackdump mcp -new claude-code ~/my-slack-project
```

After running this command:

1. `~/my-slack-project/.mcp.json` registers the Slackdump MCP server for
   Claude Code (project-scoped).
2. `~/my-slack-project/CLAUDE.md` contains the main Slackdump guidance that
   Claude Code loads automatically.
3. `~/my-slack-project/.claude/` contains supplementary reference files for
   source types and direct SQLite access.

Open the project directory in Claude Code:

```
claude ~/my-slack-project
```

**Example — set up a GitHub Copilot (VS Code) project:**

```
slackdump mcp -new copilot ~/my-slack-project
```

After running this command:

1. `~/my-slack-project/.vscode/mcp.json` wires up the Slackdump MCP server
   for VS Code / GitHub Copilot Agent mode.
2. `~/my-slack-project/.github/copilot-instructions.md` provides always-on
   Slackdump guidance to Copilot.
3. `~/my-slack-project/.github/instructions/` contains additional
   file-scoped instruction files for source types and SQLite access.

Open the project directory in VS Code:

```
code ~/my-slack-project
```

## Transport

By default the server communicates over **stdio**, which is the standard
transport for local MCP integrations.  Pass `-transport http` to start an HTTP
server instead (useful for remote agents or multiple simultaneous clients).

When using HTTP transport the server listens on `http://HOST:PORT/mcp` (the
`/mcp` path is fixed).  The default listen address is `127.0.0.1:8483`; override
it with `-listen HOST:PORT`.

## Available MCP Tools

- **`load_source`** — Open (or switch to) a Slackdump archive at runtime.
- **`list_channels`** — List all channels in the archive.
- **`get_channel`** — Get detailed info for a channel by ID.
- **`list_users`** — List all users/members.
- **`get_messages`** — Read messages from a channel (paginated).
- **`get_thread`** — Read all replies in a thread.
- **`get_workspace_info`** — Workspace / team metadata.
- **`command_help`** — Get CLI flag help for any slackdump subcommand.

### Tool parameters

#### `load_source`

- **`path`** _(string, required)_ — Filesystem path to the archive file or
  directory.

Closes the currently open archive (if any) and opens the one at `path`.  Only
one source may be open at a time.

#### `get_channel`

- **`channel_id`** _(string, required)_ — Slack channel ID (e.g. `C01234ABCD`).

#### `get_messages`

- **`channel_id`** _(string, required)_ — Slack channel ID.
- **`limit`** _(number, optional)_ — Max messages to return (1–1000, default 100).
- **`after_ts`** _(string, optional)_ — Return only messages after this Slack
  timestamp (for pagination).

Thread reply counts are included but thread bodies are not; use `get_thread`
for those.  Messages are returned in ascending timestamp order.

#### `get_thread`

- **`channel_id`** _(string, required)_ — Slack channel ID containing the thread.
- **`thread_ts`** _(string, required)_ — Timestamp of the parent message (Slack
  ts format, e.g. `1609459200.000001`).

#### `command_help`

- **`command`** _(string, optional)_ — Subcommand name (e.g. `archive`,
  `workspace new`). Empty returns top-level help.

## Integration approaches

There are three ways to connect an AI agent to the MCP server.

### stdio (agent-managed)

The AI client starts and stops `slackdump mcp` automatically.  Supply the
archive path in the client config to have it loaded at startup, or omit it to
let the agent call `load_source` to open an archive on demand.  Switching
archives while the agent is running requires either calling `load_source` or
editing the config and restarting the client.

### HTTP (terminal-managed) — recommended for interactive use

You start the server yourself in a terminal and the AI client connects over
HTTP.  You can supply the archive at startup or omit it and let the agent
call `load_source`.  To switch archives, call `load_source` from the agent,
or restart the server with a different path — no reconfiguration of the AI
client needed either way.

**Step 1** — start the server in a terminal (archive optional):

```
slackdump mcp -transport http /path/to/your/archive.db
```

or without an archive (agent will call `load_source`):

```
slackdump mcp -transport http
```

The server prints the endpoint address and listens on
`http://localhost:8483/mcp`.

**Step 2** — in another terminal (or tab), start your AI agent as usual
(`opencode`, `claude`, etc.).

To use a non-default port:

```
slackdump mcp -transport http -listen 127.0.0.1:9000 /path/to/your/archive.db
```

Note: a plain `GET http://localhost:8483/mcp` hangs by design — that endpoint
streams server-sent events (SSE).  Only `POST` requests carry MCP messages.

### Agent-driven source switching via `load_source`

The `load_source` tool lets an agent open or switch archives at any time
without restarting the server.  This works with both transports.

Start without an archive:

```
slackdump mcp -transport http
```

The agent opens an archive:

```
load_source(path="/path/to/your/archive.db")
```

The agent can switch to a different archive at any time:

```
load_source(path="/path/to/another/archive.db")
```

`load_source` also works when the server was started with an initial archive —
it closes the current one and opens the new one.

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

Omit the archive argument to let the agent call `load_source` to open one:

```json
{
  "mcpServers": {
    "slackdump": {
      "command": "slackdump",
      "args": ["mcp"]
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

Omit the archive argument to let the agent call `load_source` to open one:

```json
{
  "servers": {
    "slackdump": {
      "type": "stdio",
      "command": "slackdump",
      "args": ["mcp"]
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

Omit the archive argument to let the agent call `load_source` to open one:

```json
{
  "mcp": {
    "slackdump": {
      "type": "local",
      "command": ["slackdump", "mcp"]
    }
  }
}
```

To switch archives you can call `load_source` from the agent, or update the
config and restart OpenCode.

## Flags

- **`-transport`** _(default: `stdio`)_ — MCP transport: `stdio` or `http`.
- **`-listen`** _(default: `127.0.0.1:8483`)_ — Listen address when
  `-transport=http`.
- **`-new`** — Create a new AI project layout instead of starting the server.
  Supported layouts: `opencode`, `claude-code`, `copilot`.
