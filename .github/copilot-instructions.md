# Copilot Instructions for Slackdump

## Project Overview

**Slackdump** is a Go-based tool for archiving Slack workspaces without admin privileges. It can export messages, users, channels, files, and emojis, generating Slack Export archives in Standard or Mattermost formats.

**Repository**: https://github.com/rusq/slackdump
**Language**: Go 1.25.0
**Main Package**: `github.com/rusq/slackdump/v4`

## Project Structure

```
/
├── cmd/slackdump/          # Main CLI application entry point
│   └── internal/
│       ├── bootstrap/      # Session/client initialisation shared by commands
│       ├── cfg/            # Global CLI flags and config struct
│       ├── archive/        # archive subcommand
│       ├── convertcmd/     # convert subcommand
│       ├── diag/           # diagnostics subcommand
│       ├── dump/           # dump subcommand
│       ├── emoji/          # emoji subcommand
│       ├── export/         # export subcommand
│       ├── format/         # output format helpers (CLI layer)
│       ├── golang/         # custom CLI framework (based on Go's own cmd/)
│       ├── list/           # list subcommand
│       ├── man/            # built-in help/man pages
│       ├── mcp/            # mcp subcommand (CLI glue only)
│       ├── resume/         # resume subcommand
│       ├── ui/             # TUI components (Bubble Tea)
│       ├── view/           # view subcommand
│       ├── wizard/         # interactive wizard
│       └── workspace/      # workspace management subcommand
├── auth/                   # Authentication providers and methods
├── internal/               # Internal packages (not for public API)
│   ├── cache/              # Credential and workspace caching
│   ├── chunk/              # Data chunking and streaming
│   ├── client/             # Slack API client wrapper
│   ├── convert/            # Format conversion (archive, export, dump)
│   ├── edge/               # Edge API client
│   ├── fasttime/           # Fast time parsing utilities
│   ├── fixtures/           # Test fixture helpers
│   ├── format/             # Internal formatting utilities
│   ├── mcp/                # MCP server business logic (no CLI deps)
│   ├── mocks/              # Shared mocks (auth, cache, io, os)
│   ├── nametmpl/           # Filename template engine
│   ├── network/            # Network layer with retry logic
│   ├── osext/              # OS-level utilities
│   ├── primitive/          # Generic primitives
│   ├── redownload/         # Re-download logic for missing files
│   ├── structures/         # Slack data type parsing and helpers
│   ├── testutil/           # Shared test utilities
│   └── viewer/             # Built-in export viewer (HTTP, chi router)
├── source/                 # Data source abstractions (Sourcer, Resumer, etc.)
├── stream/                 # Streaming API for large datasets
├── types/                  # Public API types
├── export/                 # Export format handling
├── downloader/             # File download functionality
├── mocks/                  # Top-level shared mocks (processor, downloader)
└── processor/              # Message processors
```

## Build and Test

### Building
```bash
# Build the CLI
go build -o slackdump ./cmd/slackdump

# Build with version info (used in releases)
make  # or make all

# Cross-platform builds
make dist
```

### Testing
```bash
# Run all tests with race detection and coverage
make test
# or directly:
go test -race -cover ./...
```

### Code Generation
```bash
# Install required tools (mockgen, stringer)
make install_tools

# Generate mocks and string methods
make generate
# or directly:
go generate ./...
```

## Coding Conventions

### Error Handling
- **Sentinel errors**: Prefix with `Err` (e.g., `ErrNoUserCache`, `ErrNotStarted`)
- **Wrapped errors**: Use `fmt.Errorf("context: %w", err)` for error wrapping
- Public API errors should provide clear context

### Naming Conventions
- **Packages**: Short, lowercase, no underscores (e.g., `network`, `chunk`, `auth`)
- **Interfaces**: Often suffixed with `er` (e.g., `SlackClienter`, `Sourcer`)
- **Files**: Group related functionality (e.g., `channels.go`, `users.go`, `messages.go`)

### Code Style
- **Comments**: Top-level package files often start with `// In this file: <description>`
- **Function comments**: Follow Go doc conventions — start with function name
- **Unexported helpers**: Prefer short, descriptive names
- **Line length**: No strict limit, but keep reasonable (~100-120 chars)

### Testing
- **Test files**: Use `_test.go` suffix, same package or `_test` package
- **Mocks**: Generated with `go:generate mockgen` (`go.uber.org/mock/mockgen`), stored in `mock_<pkg>/` subdirectories alongside the source
- **Table-driven tests**: Common pattern for multiple test cases
- **Coverage**: Aim for good coverage on public API and critical paths

## Dependencies

### Key Libraries
- **Slack API**: `github.com/rusq/slack` (fork of slack-go/slack)
- **Authentication**: `github.com/rusq/slackauth`
- **Browser automation**: `github.com/go-rod/rod` (primary); `github.com/playwright-community/playwright-go` (deprecated, kept for compatibility)
- **CLI/TUI**: `github.com/charmbracelet/bubbletea`, `huh`, `lipgloss`, `bubbles`
- **HTTP routing**: `github.com/go-chi/chi/v5` (used in the built-in viewer)
- **Filesystem adapter**: `github.com/rusq/fsadapter`
- **Database**: `modernc.org/sqlite` (for archive format)
- **Network**: `golang.org/x/time/rate` for rate limiting
- **MCP**: `github.com/mark3labs/mcp-go`
- **Testing**: `github.com/stretchr/testify`, `go.uber.org/mock`

### Module Versioning
- Current version: v4.x
- Import path: `github.com/rusq/slackdump/v4`

## API Design

### Public API (root package)
- **Session**: Main entry point for library use
- **Options pattern**: Use functional options (`WithFilesystem`, `WithLogger`, etc.)
- **Context-aware**: All long-running operations accept `context.Context`
- **Streaming**: Prefer streaming APIs for large datasets (e.g., `StreamChannels`)

### Internal packages
- NOT part of public API — can change without notice
- Use for implementation details, utilities, and CLI-specific code
- Keep internal packages focused and single-purpose

## Logging
- Uses **log/slog** (Go standard library)
- Default logger: `slog.Default()`
- Set custom logger via `WithLogger()` option or `slog.SetDefault()`
- Log levels: DEBUG, INFO, WARN, ERROR

## Rate Limiting and Retries

### Network Layer (`internal/network`)
- **Retry logic**: Automatic retries for transient errors (default: 3 attempts)
- **Wait strategy**: Cubic backoff for API errors, exponential for network errors
- **Rate limiting**: Uses `golang.org/x/time/rate.Limiter`
- **Configurable limits**: Set via `network.Limits` and `WithLimits()` option

### Best Practices
- Always use the network package's retry-aware functions
- Don't implement your own retry logic — use `network.WithRetry()`
- Respect Slack's rate limits via `Limits` configuration

## File Organization

### One concept per file
- `channels.go` — channel/conversation operations
- `users.go` — user-related operations
- `messages.go` — message retrieval
- `thread.go` — thread handling
- `config.go` — configuration types

### Test organization
- Tests in same directory as code
- Mock interfaces via `go:generate mockgen`
- Shared test utilities in `internal/testutil/`

## Common Patterns

### Options Pattern
```go
type Option func(*Session)

func WithFilesystem(fs fsadapter.FS) Option {
    return func(s *Session) {
        if fs != nil {
            s.fs = fs
        }
    }
}
```

### Error Wrapping
```go
if err != nil {
    return fmt.Errorf("failed to get channels: %w", err)
}
```

### Context Propagation
```go
func (s *Session) GetChannels(ctx context.Context, chanTypes ...string) (types.Channels, error) {
    // Always pass context through
    channels, err := s.client.GetConversationsContext(ctx, params)
    // ...
}
```

### Streaming Pattern
```go
func (s *Session) StreamChannels(ctx context.Context, chanTypes ...string) (<-chan types.Channel, <-chan error) {
    // Return channels for streaming results
}
```

## CLI Structure (`cmd/slackdump`)

### Command Organization
- Uses a **custom command framework** modelled after Go's own `cmd/go` (not Cobra) — see `cmd/slackdump/internal/golang/base/`
- Each subcommand is a `*base.Command` with `Run`, `Wizard`, `UsageLine`, `Short`, `Long`, and a `flag.FlagSet`
- Commands in `cmd/slackdump/internal/`
- Subpackages: `workspace`, `export`, `dump`, `list`, `archive`, `emoji`, `diag`, `mcp`, `view`, `resume`, `convertcmd`

### UI Components
- **TUI**: Bubble Tea based interactive UI (`internal/ui/bubbles/`)
- **Forms**: Huh library for forms (`charmbracelet/huh`)
- **Progress**: Custom progress bars and spinners

## Authentication (`auth/`)

### Supported Methods
1. **Rod browser automation** (`NewRODAuth`) — primary automated login via `go-rod`
2. **Playwright browser automation** (`NewPlaywrightAuth`) — **deprecated**, use Rod instead
3. **Token + cookie file** (`NewCookieFileAuth`) — direct token with a Netscape cookie file
4. **Value auth** (`NewValueAuth`, `NewValueCookiesAuth`) — programmatic token/cookie provider
5. **Environment variables** — `SLACK_TOKEN` and `SLACK_COOKIE`

### Provider Interface
```go
type Provider interface {
    SlackToken() string
    Cookies() []http.Cookie
    Validate() error
    Test(ctx context.Context) (*slack.AuthTestResponse, error)
}
```

## Data Formats

### Archive Format (v4)
- SQLite-based storage
- Minimal memory footprint
- Universal structure — can convert to other formats
- Default format for all operations

### Export Format
- Compatible with Slack's official export
- Two modes: Standard (legacy) and Mattermost
- ZIP archive with JSON files

### Dump Format
- One channel per file
- No workspace metadata
- Simpler, lightweight format

## Security and Privacy

### Token Handling
- Never log tokens or cookies
- `internal/structures` contains Slack data type helpers — check there for any sanitisation utilities

### Enterprise Workspaces
- May trigger security alerts
- Use `WithForceEnterprise(true)` when needed
- Document in user-facing features

## MCP Server (`internal/mcp`)

The MCP server exposes Slackdump archive data to AI agents via the Model Context Protocol.

### Two-layer architecture

| Layer | Path | Responsibility |
|---|---|---|
| Business logic | `internal/mcp/` | `Server` struct, tools, handlers — no CLI deps |
| CLI glue | `cmd/slackdump/internal/mcp/` | Wires auth/config, calls `internal/mcp.New()` |

### Key types

| Symbol | Location | Notes |
|---|---|---|
| `Server` | `internal/mcp/server.go` | Core server struct |
| `Option` | `internal/mcp/server.go` | `func(*Server)` functional option type |
| `SourceLoader` | `internal/mcp/server.go` | `func(ctx, path) (SourceResumeCloser, error)` |
| `source.SourceResumeCloser` | `source/source.go` | Composite interface: `Sourcer + Resumer + io.Closer` |
| `MockSourceResumeCloser` | `source/mock_source/mock_source.go` | Mock for tests |

### MCP library

Uses `github.com/mark3labs/mcp-go` — imported as `mcplib` (types/tool definitions) and `mcpsrv` (server) in the MCP packages.

### `New()` — functional options pattern

```go
// All options are optional; src may be nil at startup.
srv := mcp.New(
    mcp.WithSource(src),       // open a source immediately
    mcp.WithLogger(lg),        // custom slog.Logger
    mcp.WithSourceLoader(fn),  // override how load_source opens archives
)
```

If `WithSource` is not provided, the server starts with `src == nil`. Every data tool returns a helpful error directing the agent to call `load_source` first.

### Thread safety

`Server.src` is guarded by `sync.RWMutex`:
- **Read lock** — all tool handlers call `s.source()` (the read-locked getter).
- **Write lock** — `s.loadSource()` closes the old source then swaps in the new one.

`loadSource()` always swaps even if `Close()` on the old source returns an error (the error is logged as WARN).

### `load_source` tool

Registered as the first tool in `tools()`. Takes a single `path` string argument (filesystem path to a Slackdump archive). On success it logs the archive type and returns a human-readable summary.

### Adding a new MCP tool

1. Add a `toolXxx() mcplib.Tool` method and `handleXxx()` handler method to `tools.go`.
2. Call `s.source()` at the top of the handler; return `resultErr(errNoSource)` if nil.
3. Register the tool by appending `s.toolXxx()` in the `tools()` slice in `server.go`.
4. Add tests in `tools_test.go` — use `newTestServer()` which returns `(*Server, *mock_source.MockSourceResumeCloser)`.

### Mock generation

`source/source.go` has:
```go
//go:generate mockgen -destination=mock_source/mock_source.go . Sourcer,Resumer,Storage,SourceResumeCloser
```

After changing interfaces in `source/source.go`, regenerate with:
```bash
go generate ./source/...
```

The LSP index may lag behind after regeneration — use `go build ./...` to verify there are no real compile errors.

## Documentation

### User Documentation
- RST format in `doc/` directory
- Built-in help via `slackdump help <topic>`
- Man page: `slackdump.1`

### Command help pages
Each CLI subcommand has a Markdown help page embedded at compile time via
`//go:embed`.  The convention is:

```
cmd/slackdump/internal/<command>/assets/<command>.md
```

For example: `cmd/slackdump/internal/mcp/assets/mcp.md`

The Markdown file is assigned to the `Long` field of the `*base.Command`
struct, which is what `slackdump help <command>` prints.  When adding or
modifying a subcommand, keep its `.md` file in sync with the actual flags,
arguments, and behaviour.  The `## Flags` section at the bottom of each help
page should list every flag registered on `cmd.Flag`.

### Code Documentation
- GoDoc comments on all exported types/functions
- Package docs at top of main file
- Examples in test files or separate `example_test.go`

## Contributing Guidelines

### Before Making Changes
1. Check existing issues and discussions
2. Run tests: `make test`
3. Run code generation if needed: `make generate`
4. Follow existing code style and patterns

### Pull Request Checklist
- [ ] Tests pass (`make test`)
- [ ] Code is formatted (`gofmt`)
- [ ] New code has tests
- [ ] Documentation updated if needed
- [ ] No new linter warnings

## Common Tasks for Agents

### Adding a new API method
1. Add method to `Session` type in appropriate file (e.g., `channels.go`)
2. Use context-aware Slack client methods
3. Wrap errors with context
4. Add tests in corresponding `*_test.go`
5. Update GoDoc comments

### Adding a new CLI command
1. Create package in `cmd/slackdump/internal/`
2. Implement `*base.Command` with `Run`, `UsageLine`, `Short`, `Long`
3. Wire up in main command router
4. Add help text and documentation

### Modifying internal packages
- Safe to change — not part of public API
- Ensure no unintended dependencies from public API
- Update tests

### Working with tests
- Use `testify/assert` and `testify/require` for assertions
- Mock external dependencies with `go.uber.org/mock/mockgen`
- Use `testutil` helpers when available

## Additional Notes

- **Telegram community**: https://t.me/slackdump
- **License**: Check LICENSE file
- **Code of Conduct**: See CODE_OF_CONDUCT.md
- **Releases**: Managed via GoReleaser (`.goreleaser.yaml`)
- **CI/CD**: GitHub Actions (`.github/workflows/`)

## Troubleshooting

### Common Issues
- **invalid_auth error**: Re-authenticate via `slackdump workspace new`
- **Rate limits**: Adjust limits via `WithLimits()` option
- **Free workspace limitations**: Cannot access data older than 90 days (Slack API limitation)

### Debug Mode
- Set log level to DEBUG via slog configuration
- Use tracing (`runtime/trace`) for performance analysis
- Check network retries in logs

---

**Last Updated**: 2026-02-25
**For**: Slackdump v4.x
