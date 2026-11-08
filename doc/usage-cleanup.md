# Database Cleanup

The `cleanup` tool removes residual rows that belong to unfinished database
sessions.

## Why use cleanup?

If `slackdump` stops before a run completes, the database keeps a `SESSION`
record with `FINISHED = 0`. Those incomplete sessions can leave partial data
behind. For each unfinished session, `cleanup` removes:

- the payload rows stored in the session's chunks (`MESSAGE`, `CHANNEL`,
  `FILE`, `WORKSPACE`, `S_USER`, `CHANNEL_USER`, `SEARCH_MESSAGE`,
  `SEARCH_FILE`);
- the session's chunks;
- the unfinished session row itself.

`cleanup` also removes **orphaned payload rows**: rows whose chunk no longer
exists. Slackdump v4.5.4 and earlier could leave these behind after a cleanup,
so running `cleanup` again with a newer version repairs such databases.

Cleanup does not shrink the database file. To reclaim disk space, run
`VACUUM` afterwards:

```bash
sqlite3 /path/to/archive/slackdump.sqlite VACUUM
```

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
| `-execute` | Required flag to actually remove unfinished session data and orphaned rows |

## Example

```bash
$ slackdump tools cleanup ./slackdump_20241231_150405
Unfinished sessions: 2
Chunks in unfinished sessions: 19
Payload rows in unfinished sessions: 1210
  MESSAGE:        1187
  CHANNEL:        12
  FILE:           9
  WORKSPACE:      2
Orphaned payload rows: 0

Run with -execute to perform cleanup.

$ slackdump tools cleanup -execute ./slackdump_20241231_150405
Unfinished sessions: 2
Chunks in unfinished sessions: 19
Payload rows in unfinished sessions: 1210
  MESSAGE:        1187
  CHANNEL:        12
  FILE:           9
  WORKSPACE:      2
Orphaned payload rows: 0

Removed sessions: 2
Removed chunks: 19
Removed payload rows: 1210
  MESSAGE:        1187
  CHANNEL:        12
  FILE:           9
  WORKSPACE:      2
Removed orphaned payload rows: 0

To reclaim disk space, run VACUUM on the database, i.e.:
    sqlite3 "slackdump_20241231_150405/slackdump.sqlite" VACUUM
```
