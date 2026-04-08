# Database Dedupe

The `dedupe` tool removes identical duplicate messages, users, channels,
channel users, and files created by resume look-back overlap.

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
```

Pass the archive directory that contains `slackdump.sqlite`. You do not need to
point the command at the database file itself.

## Flags

| Flag | Description |
|------|-------------|
| `-execute` | Required flag to actually remove duplicate entities |

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
