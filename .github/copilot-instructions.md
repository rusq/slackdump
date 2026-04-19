# Copilot Instructions for Slackdump

## Project Overview

**Slackdump** is a Go-based tool for archiving Slack workspaces without admin privileges. It can dump messages, export Slack-compatible archives, build SQLite-backed archives, resume previous runs, search archives, run a local MCP server, and render archives in a built-in browser viewer or as static HTML.

- Repository: `https://github.com/rusq/slackdump`
- Module path: `github.com/rusq/slackdump/v4`
- Language: Go `1.25.0`
- Main CLI entry point: `./cmd/slackdump`

## Project Structure

```text
/
├── cmd/slackdump/                 # Main CLI application
│   └── internal/
│       ├── apiconfig/             # API config validation and setup
│       ├── archive/               # archive and search commands
│       ├── bootstrap/             # shared CLI startup helpers
│       ├── cfg/                   # global CLI flags and config
│       ├── convertcmd/            # convert subcommand
│       ├── diag/                  # "slackdump tools" utilities
│       ├── dump/                  # dump subcommand
│       ├── emoji/                 # emoji subcommand
│       ├── export/                # export subcommand
│       ├── format/                # formatting helpers for CLI output
│       ├── golang/                # internal command framework
│       ├── list/                  # list users/channels subcommands
│       ├── man/                   # embedded help/man pages
│       ├── mcp/                   # mcp server CLI and project scaffolding
│       ├── resume/                # resume archive command
│       ├── ui/                    # TUI helpers and widgets
│       ├── view/                  # browser viewer command
│       ├── wizard/                # interactive wizard
│       └── workspace/             # saved workspace management
├── auth/                          # auth providers, browser flows, UI
├── downloader/                    # file download support
├── export/                        # export/message formatting helpers
├── internal/
│   ├── cache/                     # auth, user, and channel caches
│   ├── chunk/                     # chunk directory and database backends
│   ├── client/                    # Slack API client wrappers
│   ├── convert/                   # archive/export/dump/html conversion logic
│   ├── edge/                      # Slack Edge/Web API helpers, canvas support
│   ├── fixtures/                  # test fixtures
│   ├── format/                    # shared output formatting
│   ├── mcp/                       # MCP server and tools
│   ├── mocks/                     # shared mocks
│   ├── nametmpl/                  # filename templating
│   ├── network/                   # retry and rate-limit helpers
│   ├── osext/                     # OS-level utilities
│   ├── primitive/                 # shared primitives
│   ├── redownload/                # repair missing downloaded files
│   ├── structures/                # parsed Slack entities and helpers
│   └── viewer/                    # built-in viewer and static renderer
├── mocks/                         # top-level mocks
├── processor/                     # message processors
├── source/                        # archive/export/dump/database source abstraction
├── stream/                        # streaming APIs and caches
└── types/                         # public types
```

## Current CLI Surface

Important command groups:

- `slackdump archive`
- `slackdump dump`
- `slackdump export`
- `slackdump convert`
- `slackdump resume`
- `slackdump view`
- `slackdump search`
- `slackdump emoji`
- `slackdump list users`
- `slackdump list channels`
- `slackdump workspace ...`
- `slackdump config ...`
- `slackdump mcp`
- `slackdump tools ...`

Notable newer functionality that should be reflected in code suggestions:

- `slackdump convert` supports `chunk`, `database`, `dump`, `export`, and `html`.
- `slackdump convert` supports `-dm-mode` for single-user vs multi-user DM export conversion.
- `slackdump tools cleanup` removes unfinished database-session residue.
- `slackdump tools dedupe` removes duplicate rows created by resume overlap.
- `slackdump tools merge` merges multiple archive databases.
- `slackdump mcp -new <layout>` scaffolds AI-tool project layouts for `opencode`, `claude-code`, and `copilot`.
- Viewer and static HTML rendering support richer routing and canvas rendering.

## Build and Test

### Building

```bash
# Standard build
go build -o slackdump ./cmd/slackdump

# Preferred repo build target with version metadata
make all

# Debug build
make debug

# Cross-platform archives
make dist
```

### Testing

```bash
# Primary test target
make test

# Direct equivalent
go test -race -cover ./...

# Full local verification suite
make test-all
```

### Code Generation and Checks

```bash
make install_tools
make generate
make vet
make lint
```

## Coding Conventions

### Error Handling

- Prefer wrapped errors with context: `fmt.Errorf("open archive: %w", err)`.
- Sentinel errors use the `Err...` convention.
- For optional capabilities on shared interfaces, prefer small extension interfaces plus runtime type assertions instead of widening a widely used interface.
- When optional behavior is absent, degrade gracefully with `source.ErrNotFound`, `fs.ErrNotExist`, or `source.ErrNotSupported` as appropriate.

### Interfaces and Compatibility

- `source.Sourcer` is a core cross-format abstraction used by viewer, conversion, and MCP layers.
- Avoid adding methods to `source.Sourcer` unless a breaking change is intentional.
- If a consumer needs extra behavior, define an unexported extension interface in the consumer package and type-assert.

### Logging

- Use `log/slog`.
- CLI code typically uses `cfg.Log`.
- Library code defaults to `slog.Default()` unless a logger is injected.
- Prefer structured logging with stable keys.

### Style

- Follow normal Go naming and doc-comment conventions.
- Keep packages focused and small.
- Table-driven tests are common.
- Generated mocks use `go:generate mockgen`.

## Architecture Notes

### Source and Conversion

- `source.Load` auto-detects chunk directories, dump directories/zips, Slack export directories/zips, and SQLite databases.
- `internal/convert` contains conversion logic; CLI flag handling belongs in `cmd/slackdump/internal/convertcmd`.
- Static HTML export is implemented through the viewer renderer, not via a separate rendering stack.

### Database-Backed Archives

- SQLite archives are a primary format and support resume, cleanup, dedupe, merge, and search workflows.
- Repository migrations live under `internal/chunk/backend/dbase/repository/migrations/`.
- Database tools often operate on an archive directory containing `slackdump.sqlite`, not directly on the DB file path.

### Viewer and Rendering

- The built-in viewer lives in `internal/viewer`.
- It supports archive, export, and dump sources.
- Static HTML conversion reuses viewer rendering in `renderer.ModeStatic`.
- Recent viewer work includes aliases, improved routes, file handling, and canvas rendering.

### MCP

- CLI setup for MCP is in `cmd/slackdump/internal/mcp`.
- Server/tool implementations are in `internal/mcp`.
- The MCP server can start with or without an archive; if no archive is provided, clients should call `load_source`.
- Supported MCP tools include `load_source`, `list_channels`, `get_channel`, `list_users`, `get_messages`, `get_thread`, `get_workspace_info`, and CLI-level `command_help`.

## Dependencies

Key libraries in active use:

- Slack API: `github.com/rusq/slack`
- Auth: `github.com/rusq/slackauth`
- Browser automation: `github.com/go-rod/rod`
- Playwright compatibility remains in tree for compatibility paths
- CLI/TUI: `bubbletea`, `bubbles`, `huh`, `lipgloss`
- MCP: `github.com/mark3labs/mcp-go`
- HTTP router: `github.com/go-chi/chi/v5`
- SQLite: `modernc.org/sqlite`
- Markdown/rendering: `github.com/yuin/goldmark`
- Testing: `testify`, `go.uber.org/mock`

## Guidance for Suggestions

- Keep CLI parsing and business logic separated; put command wiring in `cmd/slackdump/internal/...`, reusable logic in root or `internal/...` packages.
- Preserve support for multiple source types when changing viewer, search, convert, or MCP code.
- Be careful with resume-related changes; overlap is intentional and downstream dedupe tools handle cleanup.
- Prefer extending existing helpers in `bootstrap`, `source`, `internal/convert`, `internal/viewer`, and `internal/network` instead of adding parallel implementations.
- When changing user-visible commands or flags, update embedded help/docs under `cmd/slackdump/internal/**/assets/` and `doc/` as needed.
