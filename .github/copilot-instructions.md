# Copilot Instructions for Slackdump

## Project Overview

**Slackdump** is a Go-based tool for archiving Slack workspaces without admin privileges. It can export messages, users, channels, files, and emojis, generating Slack Export archives in Standard or Mattermost formats.

**Repository**: https://github.com/rusq/slackdump
**Language**: Go 1.24.2
**Main Package**: `github.com/rusq/slackdump/v3`

## Project Structure

```
/
├── cmd/slackdump/          # Main CLI application entry point
├── auth/                   # Authentication providers and methods
├── internal/               # Internal packages (not for public API)
│   ├── cache/             # User and channel caching
│   ├── chunk/             # Data chunking and streaming
│   ├── client/            # Slack API client wrapper
│   ├── convert/           # Format conversion (archive, export, dump)
│   ├── edge/              # Edge API client
│   ├── network/           # Network layer with retry logic
│   ├── structures/        # Slack data type parsing
│   └── viewer/            # Built-in export viewer
├── source/                 # Data source abstractions
├── stream/                 # Streaming API for large datasets
├── types/                  # Public API types
├── export/                 # Export format handling
├── downloader/            # File download functionality
└── processor/             # Message processors
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
- **Interfaces**: Often suffixed with `er` (e.g., `SlackClienter`, `sourcer`)
- **Files**: Group related functionality (e.g., `channels.go`, `users.go`, `messages.go`)

### Code Style
- **Comments**: Top-level package files start with `// In this file: <description>`
- **Function comments**: Follow Go doc conventions - start with function name
- **Unexported helpers**: Prefer short, descriptive names
- **Line length**: No strict limit, but keep reasonable (~100-120 chars)

### Testing
- **Test files**: Use `_test.go` suffix, same package or `_test` package
- **Mocks**: Generated with `go:generate mockgen`, stored alongside or in `mocks_test.go`
- **Table-driven tests**: Common pattern for multiple test cases
- **Coverage**: Aim for good coverage on public API and critical paths

## Dependencies

### Key Libraries
- **Slack API**: `github.com/rusq/slack` (fork of slack-go/slack)
- **Authentication**: `github.com/rusq/slackauth`
- **CLI/TUI**: `github.com/charmbracelet/bubbletea`, `huh`, `lipgloss`, `bubbles`
- **Database**: `modernc.org/sqlite` (for archive format)
- **Network**: `golang.org/x/time/rate` for rate limiting
- **Testing**: `github.com/stretchr/testify`, `go.uber.org/mock`

### Module Versioning
- Current version: v3.x
- Import path: `github.com/rusq/slackdump/v3`
- v3.1.2 is retracted (broken build)

## API Design

### Public API (root package)
- **Session**: Main entry point for library use
- **Options pattern**: Use functional options (`WithFilesystem`, `WithLogger`, etc.)
- **Context-aware**: All long-running operations accept `context.Context`
- **Streaming**: Prefer streaming APIs for large datasets (e.g., `StreamChannels`)

### Internal packages
- NOT part of public API - can change without notice
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
- Don't implement your own retry logic - use `network.WithRetry()`
- Respect Slack's rate limits via `Limits` configuration

## File Organization

### One concept per file
- `channels.go` - channel/conversation operations
- `users.go` - user-related operations
- `messages.go` - message retrieval
- `thread.go` - thread handling
- `config.go` - configuration types

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
- Uses **cobra-like** command structure (custom implementation)
- Commands in `cmd/slackdump/internal/`
- Subpackages: `workspace`, `export`, `dump`, `list`, `archive`, `emoji`, `diag`

### UI Components
- **TUI**: Bubble Tea based interactive UI (`internal/ui/bubbles/`)
- **Forms**: Huh library for forms (`charmbracelet/huh`)
- **Progress**: Custom progress bars and spinners

## Authentication (`auth/`)

### Supported Methods
1. **EZ-Login 3000**: Automated browser-based login (Playwright/Rod)
2. **Manual token/cookie**: Direct token and cookie input
3. **Environment variables**: `SLACK_TOKEN`, `SLACK_COOKIE`
4. **Value auth**: Programmatic token/cookie provider

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

### Archive Format (v3)
- SQLite-based storage
- Minimal memory footprint
- Universal structure - can convert to other formats
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
- Use `structures.RedactTokens()` for sanitization
- Patterns in `internal/structures/structures.go`

### Enterprise Workspaces
- May trigger security alerts
- Use `WithForceEnterprise(true)` when needed
- Document in user-facing features

## Documentation

### User Documentation
- RST format in `doc/` directory
- Built-in help via `slackdump help <topic>`
- Man page: `slackdump.1`

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
2. Implement command logic
3. Wire up in main command router
4. Add help text and documentation

### Modifying internal packages
- Safe to change - not part of public API
- Ensure no unintended dependencies from public API
- Update tests

### Working with tests
- Use `testify/assert` and `testify/require` for assertions
- Mock external dependencies with `mockgen`
- Use `testutil` helpers when available

## Additional Notes

- **Telegram community**: https://t.me/slackdump
- **License**: Check LICENSE file
- **Code of Conduct**: See CODE_OF_CONDUCT.md
- **Releases**: Managed via GoReleaser (`.goreleaser.yaml`)
- **CI/CD**: GitHub Actions (`.github/workflows/`)

## Troubleshooting

### Common Issues
- **invalid_auth error**: Re-authenticate with `slackdump workspace new`
- **Rate limits**: Adjust limits via `WithLimits()` option
- **Free workspace limitations**: Cannot access data older than 90 days (Slack API limitation)

### Debug Mode
- Set log level to DEBUG via slog configuration
- Use tracing (`runtime/trace`) for performance analysis
- Check network retries in logs

---

**Last Updated**: 2026-01-29
**For**: Slackdump v3.x
