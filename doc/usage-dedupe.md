# Database Dedupe

The `dedupe` tool removes identical duplicate messages, users, channels,
channel users, and files created by resume look-back overlap. It can also
collapse messages by channel and timestamp for merged exports where Slack
regenerated volatile message fields between exports.

## Why use dedupe?

`slackdump resume` intentionally looks back in time so it can pick up new
replies on older messages. That overlap can record the same payload in more
than one session. `dedupe` removes the older identical copies, keeps the
latest copy, and prunes chunks that become empty as a result.

## Usage

```bash
# Preview what would be removed
slackdump tools dedupe /path/to/archive

# Actually perform dedupe
slackdump tools dedupe -execute /path/to/archive

# Collapse message rows by channel and timestamp
slackdump tools dedupe -mode message-key -execute /path/to/archive
```

Pass the archive directory that contains `slackdump.sqlite`. You do not need to
point the command at the database file itself.

## Flags

| Flag | Description |
|------|-------------|
| `-execute` | Required flag to actually remove duplicate entities |
| `-mode` | Message dedupe mode: `exact` keeps current byte-for-byte behavior; `message-key` keeps the latest row per channel and timestamp |

`message-key` is useful after merging multiple Slack exports of the same
workspace when duplicate messages differ only in volatile Slack-generated JSON
fields, such as `blocks[].block_id`. It is intentionally opt-in because it can
discard older edited or reaction variants for the same message timestamp. It
also treats channel-history and thread-message copies of the same channel
timestamp as one logical message and keeps the latest database row.

## Example

```bash
$ slackdump tools dedupe ./slackdump_20241231_150405
Duplicate messages: 42
Duplicate users: 548
Duplicate channels: 3
Duplicate channel users: 17
Duplicate files: 1
Chunks to prune: 14

Run with -execute to perform dedupe.

$ slackdump tools dedupe -execute ./slackdump_20241231_150405
Duplicate messages: 42
Duplicate users: 548
Duplicate channels: 3
Duplicate channel users: 17
Duplicate files: 1
Chunks to prune: 14

Removed messages: 42
Removed users: 548
Removed channels: 3
Removed channel users: 17
Removed files: 1
Removed chunks: 14
```
