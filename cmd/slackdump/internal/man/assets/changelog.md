# What's New?

## v4.5.0

### New Features

- **More flexible database deduplication**: `slackdump tools dedupe` now
  supports `-mode message-key`, which collapses messages by channel and
  timestamp. This is useful after merging exports where Slack regenerated
  volatile JSON fields. It remains opt-in: it keeps the latest row and can
  discard older edited or reaction variants with the same timestamp.

- **Codex MCP project scaffolding**: `slackdump mcp -new codex <directory>`
  creates a project-scoped `.codex/config.toml` and bundled Slackdump skills.
  The Copilot scaffold and documentation now use VS Code's `.mcp.json` and
  `mcpServers` configuration format.

- **Viewer landing preference**: the live viewer now offers a Settings control
  to open conversations at the oldest message (the default) or the latest
  message, plus a one-time **Jump to latest** action. The preference is stored
  in browser local storage and never changes the archive or static HTML output.

### Security and Upgrade Notes

The security fixes in this release were informed by a security report compiled
by Cornelius Roemer. We thank Cornelius for the analysis that served as the
basis for this work.

- **Viewer host protection**: `slackdump view` now accepts only localhost,
  loopback hosts, and the host named by `-listen` by default. If you expose the
  viewer through a LAN address or reverse proxy, repeat
  `-allow-host <hostname-or-ip>` for each expected Host header.

- **Safer viewer and static HTML output**: viewer responses use security
  headers, archived Canvas markup is sanitized and sandboxed, and static
  Canvas attachments are written as sanitized documents.

- **Fail-closed self-update verification**: `slackdump tools update` now
  requires one valid, unambiguous SHA-256 entry in `checksums.txt` before it
  extracts a downloaded release asset. Release publishers must include that
  file and an entry for every installable asset.

- **Export credential warning**: when `-export-token` is used, Slackdump now
  warns that the token is embedded in file URLs; sharing that export also shares
  the credential's access. Download error messages also redact URL credentials.

- **Go 1.26 is now required** to build Slackdump from source.

### Bug Fixes

- Fixed a conversation pipeline shutdown deadlock and ensured cancellation
  stops queued channel and thread work cleanly.
- Fixed channel timelines so they retain empty thread parents and show the
  refreshed parent after a direct thread resume.
- `slackdump tools dedupe` now uses index-friendly predicates for non-null
  keys, preventing very slow preview runs on archives with many duplicate
  channel-user rows.
- Archiving with `-channel-users` now treats Slack's `user_not_found` response
  as non-fatal and stores de-duplicated channel membership lists.
- Export, conversion, and merge workflows now skip Slack `tombstone`,
  `hidden_by_limit`, and `external` file records instead of failing.
- Updated the Slack Edge bootstrap endpoint to `client.init`, improving
  compatibility with current Slack responses.

## v4.4.0

### New Features

- **Faster, more selective resume**: `slackdump resume` now supports
  `-skip-complete-threads`, `-skip-stale-threads <duration>`, and
  `-skip-stale-channels <duration>` to avoid re-fetching thread or channel
  entities that are already complete or have gone dormant.  Stale filters run
  before any Slack API calls and exit successfully as a no-op when everything is
  filtered out.

- **Post-resume dedupe**: `slackdump resume -dedupe` runs duplicate cleanup
  automatically after a successful resume, using the same cleanup logic as
  `slackdump tools dedupe`.

- **Improved interactive login compatibility**: browser-based workspace login
  now prefers an installed system browser for the interactive "Login in
  Browser" flow, working around Slack rejecting the bundled Chromium revision.
  Use `slackdump workspace new -bundled-browser` to force the previous bundled
  browser behaviour.

- **Viewer static and interactive polish**: the archive viewer now has shared
  viewer-side JavaScript for side-panel state, active-channel highlighting,
  connection-status handling, and tab keyboard navigation.  Static HTML output
  also supports user profile panels via anchors, and viewer layout has improved
  responsive behaviour.

- **Wizard help pager**: command wizards now expose a Help menu item backed by
  a scrollable pager, with centralised key bindings and consistent help styling
  across TUI components.

- **Replit + Google Drive backup recipe**: `contrib/replit_drive_backup`
  contains a resumable uploader that mirrors a Slackdump archive directory to
  Google Drive from a Replit workspace.

### Bug Fixes

- `thread_not_found` is now treated as a non-critical thread fetch result, so
  stale or deleted threads no longer fail a run.
- `slackdump resume -threads -skip-complete-threads` now keeps direct thread
  resume items and applies the complete-thread check after the first successful
  `conversations.replies` page.
- Resume now distinguishes an invalid empty archive from a valid no-op caused
  by stale filters, and filtering to no results no longer reports an error.
- File download deduplication during resume is now an explicit database
  controller option instead of depending on the command name.
- Edge client setup is restricted to `xoxc-` client tokens, avoiding failures
  with token types that cannot use Slack Edge APIs.
- Fixed several TUI regressions: shared Huh keymap state, config checker pane
  sizing, boolean toggles, number-key navigation, date picker polish, and
  scrollable wizard help sizing.

## v4.3.0

### New Features

- **`slackdump tools merge`**: merge one or more Slackdump sources (archives,
  exports, or databases) from the same workspace into an existing database
  archive.  Pass `-check` to verify compatibility without modifying the target,
  `-files=false` to skip file attachments, or `-avatars=false` to skip user
  avatars.  Merge does not deduplicate — run `slackdump tools dedupe` on the
  result if sources overlap.

- **`slackdump tools cleanup`**: remove residual rows that belong to unfinished
  database sessions.  Run without flags to preview, add `-execute` to commit the
  removal.

- **`slackdump tools dedupe`**: remove identical duplicate messages, users,
  channels, channel users, and files introduced by resume look-back overlap.
  Run without flags to preview, add `-execute` to commit.

- **`-dm-mode` flag for `slackdump convert`**: controls how 1:1 DM channels are
  serialised into `dms.json` during export conversion.  `single` (default)
  preserves the historic single-user layout; `multi` uses the observed IM
  membership list, which is the correct choice when converting a merged
  multi-user archive.

### Bug Fixes

- Fixed a regression in user ordering during export conversion.
- Fixed range copy bug, DM member assignment, and SQLite variable limit
  in migration logic (contributed by Fizmatik).
- Improved FILE-type attachment deduplication matching.
- `slackdump resume` now returns a clear error when the target database does
  not exist instead of silently creating an empty one.
- `slackdump resume` now returns a clear error when the target archive contains
  no sessions.

## v4.1.0

### New Features

- **MCP Server** (`slackdump mcp`): a read-only Model Context Protocol server
  for querying Slackdump archives with AI agents (Claude, GitHub Copilot,
  OpenCode, etc.).  Supports both **stdio** and **HTTP** transports.  Available
  tools: `load_source`, `list_channels`, `get_channel`, `list_users`,
  `get_messages`, `get_thread`, `get_workspace_info`, and `command_help`.

- **Auto-updater** (`slackdump tools update`): checks for a newer release on
  GitHub and optionally installs it.  Pass `-auto` to update without prompts.
  Marked experimental; supports brew, apt, pacman, and direct binary replacement.

- **`-fail-hard` flag**: opt in to hard-failing on non-critical per-channel
  errors (e.g. `not_in_channel`, `channel_not_found`) across `archive`,
  `export`, and `dump`.  By default these errors are skipped and logged.

- **`-member-only` in `list channels`**: the `-member-only` flag is now
  respected by `slackdump list channels` in addition to `archive`.

- **QR code input size override**: the maximum size of the QR code image paste
  field can be tuned via the `QR_CODE_SIZE` environment variable for workspaces
  that produce unusually large QR images.

### Bug Fixes

- Enterprise channel filtering: fixed incorrect channel filtering that could
  silently drop channels in Enterprise Grid workspaces.

- `not_in_channel` no longer aborts a full-workspace archive; the channel is
  skipped and the run continues (see also `-fail-hard` above).

- `IsMember` logic now falls back to the `C`-prefix heuristic for channels
  where the membership field is absent, fixing missing channels in
  `--member-only` runs.

## v4.0.0

- New channel type filtering via `--chan-types` and wizard multi-select, wired through list/archive/export/resume flows.
- Optional custom profile field labels with `--custom-labels`, including UI support; uses a new user profile fetch path.
- Channel type constants now align with Slack string values; channel retrieval defaults to all types when none specified.
- Listing commands now report empty results early and expose list sizes; added tests for list length helpers.
- Internal stream/control updates for custom user profile fetching, plus expanded mocks and tests.
- Safer enum String() methods guard against negative values across generated stringers.
- License switch from GPLv3 to AGPLv3.
- Better handling of cancellation in various packages.

## v3.1.0

- Filenames in Slack Export are dated in the America/Los\_Angeles timezone to
  align with the Slack export format;
- 5x faster conversion to Slack export, when using database backend, compared to
  the chunk file backend.
- backend for export, archive and dump formats is changed to database;
- archive and search formats is changed to database;
- universal converter to export for any other format.

## v3.0.0

Gist:
- 2.6x dump speed improvement on channels with threads;
- Support for enterprise workspaces;
- json logging on demand;
- new structured CLI;
- improved TUI for the wizardry with bells and whistles;
- multiple workspaces with ability to switch between them without `-auth-reset`;
- api limits configuration files;
- uninstallation tool;
- slackdump system information tool;
- pgp encryption for traces (under tools);
- search results archival;

### New Archive format

Consider using the new `archive` command to save your workspace data.  You can read about
it in the `slackdump help archive` command and the format it produces in the
`slackdump help chunk` command.

### Viewer

Slackdump V3 introduces a viewer for exported data.  To view the exported data, run:
```
slackdump view <export file or directory>
```

NOTE: search results are not supported by the viewer yet.


### Breaking changes

- `-download` flag renamed to `-files` and is set to "true" by default;
- `-r` flag that allowed to generate text files was replaced by
  `slackdump format` command.

### New features

- Completely rewritten CLI, based on `go` command source code (see
  [Licenses][1]);
- Default API limits can be overridden with configuration file (use
  `-config <file>`);
- Slack Workspaces:
    - Slackdump remembers credentials for multiple Slack Workspaces;
    - It is possible to select the "_Current_" Workspace using the
      `workspace select` command;
    - The "_Current_" workspace can be overridden by providing the `-w <name>`
      flag.
- Slackdump `archive` mode allows to dump the entire workspace into a directory
  of chunk files.
- Slackdump `convert` mode allows to convert chunk files into other formats,
  such as Slack export format, or Slackdump format.

### Changes

- Default output location (**BASE_LOC** environment variable), if not set by the
  user, defaults to the ZIP file "slackdump\_YYYYMMDD\_HHmmSS.zip", where
  `YYYYMMDD` is the current date (for example `20221103`) and `HHmmSS` is the
  current time with seconds (for example `185803`);
- To reset all authentication data (similar to old `-auth-reset`), run
  `slackdump workspace delete -a -y` where `-a` is for "all" and `-y` to
  answer "yes" to all questions;
- Flag `-user-cache-file` was removed.
- Slackdump does not cache users on each startup, to speed up execution, it
  users lazy caching. For example, when the list command is requested.  The
  user cache behaviour can still be controlled by `-no-user-cache` and
  `-user-cache-retention` flags.


## Library changes in v3.0+

### Slackdump Core

- `Options` reorganised, API limits are extracted into a Limits variable. Tier
  limits are extracted to TierLimits, and are accessible via `Limits.Tier2` and
  `Limits.Tier3` variables. Requests limits are accessible via
  `Limits.Request`.
- `Session.SetFS` method is removed, set the filesystem in `Options.Filesystem`.
- Introduced `Close()` interface method on `fsadapter.FS`.  `fsadapter.Close` is
  removed.

### Licenses

- `./cmd/internal/golang` is BSD licensed.
- Slackdump is AGPL-3 licensed.

[1]: #licenses
