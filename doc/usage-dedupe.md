# Message Dedupe

The `dedupe` tool removes identical duplicate messages created by resume
look-back overlap.

## Why use dedupe?

`slackdump resume` intentionally looks back in time so it can pick up new
replies on older messages. That overlap can record the same message payload in
more than one session. `dedupe` removes the older identical copies, keeps the
latest copy, and prunes message chunks that become empty as a result.

## Usage

```bash
# Preview what would be removed
slackdump tools dedupe /path/to/database.db

# Actually perform dedupe
slackdump tools dedupe -execute /path/to/database.db
```

## Flags

| Flag | Description |
|------|-------------|
| `-execute` | Required flag to actually remove duplicate messages |

## Example

```bash
$ slackdump tools dedupe slackdump.sqlite
Duplicate messages: 42
Message chunks to prune: 7

Run with -execute to perform dedupe.

$ slackdump tools dedupe -execute slackdump.sqlite
Duplicate messages: 42
Message chunks to prune: 7

Removed messages: 42
Removed chunks: 7
```
