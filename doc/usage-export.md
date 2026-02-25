# Creating a Slack Export

[Back to User Guide](README.md)

The `export` command saves your Slack workspace as a ZIP file compatible with
Slack's own export format.

For incremental/resumable archiving, prefer `slackdump archive`.  See the
[Archive vs Export vs Dump](#archive-vs-export-vs-dump) section below.

## Quick Start

```shell
# Export the whole workspace (Mattermost format by default)
slackdump export

# Export to a specific ZIP file
slackdump export -o my_export.zip

# Export only specific channels
slackdump export C12301120 D4012041

# Standard format (for use with slack-export-viewer)
slackdump export -type standard -o my_export.zip
```

## Export Types

| Type | Flag | Description |
|------|------|-------------|
| Mattermost (default) | `-type mattermost` | Compatible with `mmetl` / Mattermost bulk import |
| Standard | `-type standard` | Compatible with `slack-export-viewer` and other tools |

## File Structure

### Mattermost Format (default)

```
/
├── __uploads/             ← all uploaded files
│   └── F02PM6A1AUA/
│       └── Chevy.jpg
├── everyone/              ← channel "#everyone"
│   ├── 2022-01-01.json
│   └── 2022-01-04.json
├── DM12345678/            ← a DM conversation
│   └── 2022-01-04.json
├── channels.json
├── dms.json
└── users.json
```

### Standard Format

```
/
├── everyone/
│   ├── 2022-01-01.json
│   ├── 2022-01-04.json
│   └── attachments/
│       └── F02PM6A1AUA-Chevy.jpg
├── DM12345678/
│   └── 2022-01-04.json
├── channels.json
├── dms.json
└── users.json
```

## Including and Excluding Channels

Pass channel IDs or URLs as arguments.  Use `^` to exclude, `@file` for a
list file.  For the full syntax, run `slackdump help syntax`.

```shell
# Include only specific channels
slackdump export C12401724 C4812934

# Exclude a channel
slackdump export ^C123456

# Use a file list
slackdump export @channels.txt

# With time range
slackdump export C123456,2024-01-01T00:00:00,2024-12-31T23:59:59
```

## Viewing the Export

You can use the built-in viewer:

```shell
slackdump view my_export.zip
```

Or use one of these external tools:

- **[SlackLogViewer](https://github.com/thayakawa-gh/SlackLogViewer/releases)** —
  fast C++ desktop app with search; works with Export files.
- **[slack-export-viewer](https://github.com/hfaran/slack-export-viewer)** —
  web-based viewer; requires **Standard** format.

## Migrating to Mattermost

1. Export in Mattermost format:

   ```shell
   slackdump export -o my-workspace.zip
   ```

2. Download `mmetl` from the [mmetl GitHub page](https://github.com/mattermost/mmetl)
   and transform the export:

   ```shell
   ./mmetl transform slack -t YourTeamName -d bulk-export-attachments \
     -f my-workspace.zip -o mattermost_import.jsonl
   ```

3. Create the bulk import ZIP:

   ```shell
   mkdir data
   mv bulk-export-attachments data/
   zip -r bulk_import.zip data mattermost_import.jsonl
   ```

4. Upload and import into Mattermost:

   ```shell
   mmctl auth login http://your-mattermost-server
   mmctl import upload ./bulk_import.zip
   mmctl import list available     # note the file name with ID prefix
   mmctl import process <filename>
   mmctl import job list           # monitor progress
   ```

See the [Mattermost documentation](https://docs.mattermost.com/onboard/migrating-to-mattermost.html)
for full details.

## Migrating to Discord

Use [Slackord2](https://github.com/thomasloupe/Slackord2) — a GUI tool
compatible with Slackdump export files.

## Archive vs Export vs Dump

| Command | Format | Best for |
|---------|--------|----------|
| `archive` | SQLite database | Incremental backups, large workspaces, SQL queries |
| `export` | Slack-compatible ZIP | Mattermost/Discord migration, compatibility with viewers |
| `dump` | Per-channel JSON | Low-level access, custom tooling |

`archive` is the recommended default.  It is faster than `export`, can be
resumed, and can be converted to Export or Dump format afterwards:

```shell
slackdump convert -f export ./slackdump.sqlite
```

In fact, Slackdump uses a temporary archive file and then converts it to
export, when you run `slackdump export`.

## Key Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-o location` | `slackdump_<ts>.zip` | Output directory or ZIP file |
| `-type value` | `mattermost` | Export type: `mattermost` or `standard` |
| `-files` | `true` | Download file attachments |
| `-export-token string` | — | Append export token to file URLs (or set `SLACK_FILE_TOKEN` env var) |
| `-member-only` | — | Only export channels the current user is a member of |
| `-chan-types value` | all | Filter channel types (`public_channel`, `private_channel`, `im`, `mpim`) |
| `-time-from YYYY-MM-DDTHH:MM:SS` | — | Oldest message timestamp |
| `-time-to YYYY-MM-DDTHH:MM:SS` | now | Newest message timestamp |
| `-workspace name` | current | Override the active workspace |

[Back to User Guide](README.md)
