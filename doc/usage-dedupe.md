# Database Dedupe

The `dedupe` tool removes identical duplicate messages, users, channels,
channel users, and files created by resume look-back overlap. It can also
collapse messages by channel and timestamp for merged exports where Slack
regenerated volatile message fields between exports.

## Why use dedupe?

`slackdump resume` intentionally looks back in time so it can pick up new
replies on older messages. That overlap can record the same payload in more
than one session. `dedupe` removes duplicates and prunes chunks that become
empty as a result. In both modes, it keeps the copy with the highest chunk ID
from a finished session. If none of the copies came from a finished session,
it keeps the copy with the highest chunk ID overall.

## Usage

```bash
# Preview what would be removed
slackdump tools dedupe /path/to/archive

# Remove duplicates directly without a preview
slackdump tools dedupe -execute /path/to/archive

# Collapse message rows by channel and timestamp
slackdump tools dedupe -mode message-key -execute /path/to/archive
```

Pass the archive directory that contains `slackdump.sqlite`. You do not need to
point the command at the database file itself.

## Flags

| Flag | Description |
|------|-------------|
| `-execute` | Remove duplicate entities directly without first printing a preview |
| `-mode` | `exact` matches message payloads byte for byte. `message-key` matches by channel and timestamp regardless of payload. |

`message-key` is useful after merging multiple Slack exports of the same
workspace when duplicate messages differ only in volatile Slack-generated JSON
fields, such as `blocks[].block_id`. It is intentionally opt-in because it can
discard edited or reaction variants for the same message timestamp. It
also treats channel-history and thread-message copies of the same channel
timestamp as one logical message.

## Example

```bash
$ slackdump tools dedupe ./slackdump_20241231_150405
Duplicate messages: 42
Duplicate canvas messages: 0
Duplicate users: 548
Duplicate channels: 3
Duplicate channel users: 17
Duplicate files: 1
Chunks to prune: 14

Run with -execute to perform dedupe.

$ slackdump tools dedupe -execute ./slackdump_20241231_150405
Removed messages: 42
Removed canvas messages: 0
Removed users: 548
Removed channels: 3
Removed channel users: 17
Removed files: 1
Removed chunks: 14
```
