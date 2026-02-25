# Dumping Conversations

[Back to User Guide](README.md)

The `dump` command saves individual conversations or threads to JSON files.
It is the original low-level mode of Slackdump and applies minimal
transformation to the raw Slack API output.

For archiving an entire workspace, see [usage-export.md](usage-export.md) or
run `slackdump help archive`.

## Quick Start

```shell
# Dump a channel and a DM (files downloaded by default)
slackdump dump C051D4052 DHYNUJ00Y

# Dump from a list file
slackdump dump @my_channels.txt

# Dump a single thread (URL or colon notation)
slackdump dump https://ora600.slack.com/archives/C051D4052/p1665917454731419
slackdump dump C051D4052:1665917454.731419
```

On Windows, replace `./slackdump` with `slackdump` in all examples.

## Output Location

By default Slackdump writes files to a ZIP archive named
`slackdump_YYYYMMDD_HHmmSS.zip` in the current directory.

| Flag | Description |
|------|-------------|
| `-o some_dir` | Write to a directory named `some_dir` |
| `-o my_archive.zip` | Write to a specific ZIP file |

## File Attachments

Files are downloaded by default.  To disable:

```shell
slackdump dump -files=false C051D4052
```

## Specifying Channels and Threads

Provide channel IDs, conversation URLs, or thread URLs as arguments.  They can
be freely mixed:

```shell
slackdump dump C051D4052 DHYNUJ00Y @my_channels.txt \
  C051D4052:1665917454.731419
```

For the full channel/thread syntax (exclusions, time ranges, file lists), see:

```shell
slackdump help syntax
```

### Getting a Conversation URL

1. In Slack, right-click the conversation in the left pane.
2. Choose **Copy link**.

### Getting a Thread URL

1. Open the thread in Slack.
2. On the parent message, click the **⋮** (three dots) menu → **Copy link**.
3. Pass that URL to `slackdump dump`.

### Thread Colon Notation

Slackdump also accepts threads in a compact `CHANNEL:THREAD_TS` format:

```
C051D4052:1665917454.731419
```

## Time Range

Limit messages to a specific period using global flags:

```shell
slackdump dump -time-from 2024-01-01T00:00:00 -time-to 2024-03-31T23:59:59 C051D4052
```

Or per-channel using the [syntax](https://github.com/rusq/slackdump/blob/master/cmd/slackdump/internal/man/assets/syntax.md):

```shell
slackdump dump C051D4052,2024-01-01T00:00:00,2024-03-31T23:59:59
```

## Viewing the Dump

```shell
slackdump view <dump_file_or_directory>
```

Alternatively, use [Slackdump2HTML](https://github.com/kununu/slackdump2html)
to convert the dump to a browsable static HTML site.

## Converting to Other Formats

To convert a JSON dump to plain text:

```shell
slackdump format text <archive.zip or directory>
```

Run `slackdump help format` for all available formats.

## Key Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-o location` | `slackdump_<ts>.zip` | Output directory or ZIP file |
| `-files` | `true` | Download file attachments |
| `-time-from YYYY-MM-DDTHH:MM:SS` | — | Oldest message timestamp |
| `-time-to YYYY-MM-DDTHH:MM:SS` | now | Newest message timestamp |
| `-workspace name` | current | Override the active workspace |
| `-v` | — | Verbose output |

[Back to User Guide](README.md)
