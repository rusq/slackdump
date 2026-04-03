# Database Cleanup

The `cleanup` tool removes residual rows that belong to unfinished database
sessions.

## Why use cleanup?

If `slackdump` stops before a run completes, the database keeps a `SESSION`
record with `FINISHED = 0`. Those incomplete sessions can leave partial chunks
behind. `cleanup` removes the chunks that belong to unfinished sessions and
then removes the unfinished session rows themselves.

## Usage

```bash
# Preview what would be removed
slackdump tools cleanup /path/to/database.db

# Actually perform the cleanup
slackdump tools cleanup -execute /path/to/database.db
```

## Flags

| Flag | Description |
|------|-------------|
| `-execute` | Required flag to actually remove unfinished session data |

## Example

```bash
$ slackdump tools cleanup slackdump.sqlite
Unfinished sessions: 2
Chunks in unfinished sessions: 19

Run with -execute to perform cleanup.

$ slackdump tools cleanup -execute slackdump.sqlite
Unfinished sessions: 2
Chunks in unfinished sessions: 19

Removed sessions: 2
Removed chunks: 19
```
