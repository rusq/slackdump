# Archiving a Workspace

The `archive` command is the **recommended default** for backing up a Slack
workspace. It saves everything to a local SQLite database that can be queried
directly with SQL, viewed in the built-in browser viewer, or converted to other
formats later.

## Basic Usage

```bash
# Archive the entire workspace (all channels the authenticated user can see)
slackdump archive

# Archive specific channels or DMs by ID or URL
slackdump archive C01234ABCDE C98765ZYXWV
slackdump archive https://myworkspace.slack.com/archives/C01234ABCDE

# Archive channels from a file (one ID per line)
slackdump archive @channels.txt
```

By default the database is written to `slackdump_YYYYMMDD_HHMMSS.zip`
(a directory-based archive). The underlying SQLite file is
`slackdump.sqlite` inside that directory.

To choose a different output location:

```bash
slackdump archive -o /backups/myworkspace
```

## Why Use `archive` Instead of `export` or `dump`?

| Criteria | `archive` | `export` | `dump` |
|---|---|---|---|
| Format | SQLite database | Slack-compatible ZIP | Raw JSON |
| Speed | Fastest | Slower (conversion overhead) | Fast |
| Resumable | Yes (`slackdump resume`) | No | No |
| Queryable with SQL | Yes | No | No |
| Compatible with `slack-export-viewer` | Via `convert` | Yes | No |
| Suitable as source for conversion | Yes (master format) | — | — |

Use `archive` for ongoing backups. Convert to another format when you need
compatibility with a specific tool.

## Incremental Backups / Resume

If an archive run is interrupted, or you want to top it up with new messages
each day, use `slackdump resume`:

```bash
slackdump resume /backups/myworkspace
```

Resume reads the existing database to determine where each channel left off and
only fetches messages newer than the last recorded timestamp. This is far faster
than re-archiving from scratch.

See the [Troubleshooting](troubleshooting.md#resume--incremental-backups) page
if resume hangs or is unexpectedly slow.

## Selecting Which Channels to Archive

By default `archive` fetches every channel the authenticated user can access.
You can narrow this with several methods:

**Explicit channel IDs or URLs on the command line:**

```bash
slackdump archive C01234ABCDE DXXXXXXXX
```

**A file of channel IDs (`@file`):**

```
# channels.txt — one channel ID per line
C01234ABCDE
C98765ZYXWV
GABCDEFGHIJ
```

```bash
slackdump archive @channels.txt
```

To build this file, run `slackdump list channels -o channels.json`, then
extract the IDs with a tool like `jq`:

```bash
slackdump list channels -o channels.json
jq -r '.[].id' channels.json > channels.txt
```

**Only channels you are a member of:**

```bash
slackdump archive -member-only
```

**Filter by channel type:**

```bash
# Only public channels and private groups (exclude DMs)
slackdump archive -chan-types public_channel,private_channel
```

Available types: `public_channel`, `private_channel`, `im` (DMs), `mpim`
(group DMs).

## Date Range

Fetch only messages within a time window (timestamps in UTC):

```bash
slackdump archive -time-from 2024-01-01 -time-to 2024-12-31
```

The format is `YYYY-MM-DD` or `YYYY-MM-DDTHH:MM:SS`.

## File Attachments

File downloads are **enabled by default**. To disable:

```bash
slackdump archive -files=false
```

Downloaded files are placed in the `__uploads/` subdirectory of the output
location. User avatars are not downloaded by default; enable with `-avatars`.

## Key Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-o location` | auto-named `.zip` | Output directory or ZIP path |
| `-files` | `true` | Download file attachments |
| `-avatars` | `false` | Download user avatars |
| `-member-only` | `false` | Only channels the current user belongs to |
| `-chan-types` | all types | Comma-separated list of channel types to include |
| `-time-from` | (oldest) | Start of date range (UTC) |
| `-time-to` | now | End of date range (UTC) |
| `-enterprise` | `false` | Enable Enterprise Grid mode |
| `-v` | `false` | Verbose output |
| `-y` | `false` | Answer yes to all prompts (non-interactive) |
| `-limiter-boost` | (default) | Rate-limiter aggressiveness; try `0` on large workspaces |

Run `slackdump help archive` for the full flag list.

## Querying the Database

The SQLite database can be opened with any SQLite client, e.g.
[DB Browser for SQLite](https://sqlitebrowser.org/) or the `sqlite3` CLI:

```bash
sqlite3 /backups/myworkspace/slackdump.sqlite

# List all channels in the archive
SELECT id, name, is_channel, is_private FROM CHANNEL;

# Count messages per channel
SELECT channel_id, COUNT(*) AS msg_count
FROM MESSAGE
GROUP BY channel_id
ORDER BY msg_count DESC;

# Search message text
SELECT channel_id, ts, text FROM MESSAGE WHERE text LIKE '%incident%';
```

Key tables: `CHANNEL`, `MESSAGE`, `S_USER`, `FILE`, `SESSION`.

## Converting to Other Formats

Once you have a SQLite archive you can convert it to a Slack-compatible export
ZIP or the legacy chunk format:

```bash
# Convert to standard Slack export ZIP (compatible with slack-export-viewer)
slackdump convert -f export ./slackdump_20240101_000000

# Convert to Mattermost format
slackdump convert -f mattermost ./slackdump_20240101_000000/slackdump.sqlite
```

See `slackdump help convert` for all format options.

## Viewing the Archive

```bash
slackdump view ./slackdump_20240101_000000
```

This starts a local web server and opens the archive in your browser. See the
[Troubleshooting](troubleshooting.md#built-in-viewer-slackdump-view) section
if the viewer returns 404 errors for attachments.
