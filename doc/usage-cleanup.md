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
slackdump tools cleanup /path/to/archive

# Actually perform the cleanup
slackdump tools cleanup -execute /path/to/archive
```

Pass the archive directory that contains `slackdump.sqlite`. You do not need to
point the command at the database file itself.

## Flags

| Flag | Description |
|------|-------------|
| `-execute` | Required flag to actually remove unfinished session data |

## Example

```bash
$ slackdump tools cleanup ./slackdump_20241231_150405
Unfinished sessions: 2
Chunks in unfinished sessions: 19

Run with -execute to perform cleanup.

$ slackdump tools cleanup -execute ./slackdump_20241231_150405
Unfinished sessions: 2
Chunks in unfinished sessions: 19

Removed sessions: 2
Removed chunks: 19
```
