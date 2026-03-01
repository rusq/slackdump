# Database Package — Agent Reference

This document captures non-obvious behaviours, data quirks, and schema
decisions in the `dbase` package that are relevant to any agent working on or
querying the Slackdump SQLite database.

The schema is managed by [goose](https://github.com/pressly/goose) migrations
located in `repository/migrations/`.

---

## Schema Overview

### Tables

| Table          | Primary Key          | Description |
|----------------|----------------------|-------------|
| `SESSION`      | `ID` (autoincrement) | One row per `slackdump` invocation |
| `CHUNK`        | `ID` (autoincrement) | One row per Slack API response page |
| `TYPES`        | `ID`                 | Chunk type lookup (TYPE_ID → name) |
| `MESSAGE`      | `(ID, CHUNK_ID)`     | Channel and thread messages |
| `CHANNEL`      | `(ID, CHUNK_ID)`     | Slack channels / conversations |
| `FILE`         | `(ID, CHUNK_ID)`     | File attachments |
| `WORKSPACE`    | `ID` (autoincrement) | Workspace metadata |
| `S_USER`       | `(ID, CHUNK_ID)`     | Workspace members |
| `CHANNEL_USER` | `(CHANNEL_ID, USER_ID, CHUNK_ID)` | Channel membership (no DATA blob) |
| `SEARCH_MESSAGE` | `ID` (autoincrement) | Results from `slackdump search` |
| `SEARCH_FILE`  | `ID` (autoincrement) | File results from `slackdump search` |

### Views (all prefixed `V_`)

All `V_*` views are **internal to slackdump** and used to track unprocessed
threads during archiving. Do not rely on them for general analysis. They are:

| View | Purpose |
|------|---------|
| `V_CHANNEL_THREADS` | Thread count per channel/session (finished channels only) |
| `V_CHANNEL_THREAD_COUNT` | Count of actually downloaded thread parents |
| `V_UNFINISHED_CHANNELS` | Channels where thread count doesn't match downloaded |
| `V_ORPHAN_THREADS` | Threads with a parent but no downloaded children |
| `V_EMPTY_THREADS` | Threads where `LATEST_REPLY = '0000000000.000000'` |
| `V_THREAD_ONLY_THREADS` | For thread-only mode: counts parts per thread |
| `V_LATEST_MESSAGE` | Latest channel message per channel (TYPE_ID=0 only) |
| `V_LATEST_THREAD` | Latest thread message per channel+thread_ts (TYPE_ID=1 only) |

---

## Chunk Types (TYPES table)

| ID | NAME              | Stores data in   |
|----|-------------------|------------------|
|  0 | `MESSAGES`        | `MESSAGE`        |
|  1 | `THREAD_MESSAGES` | `MESSAGE`        |
|  2 | `FILES`           | `FILE`           |
|  3 | `USERS`           | `S_USER`         |
|  4 | `CHANNELS`        | `CHANNEL`        |
|  5 | `CHANNEL_INFO`    | `CHANNEL`        |
|  6 | `WORKSPACE_INFO`  | `WORKSPACE`      |
|  7 | `CHANNEL_USERS`   | `CHANNEL_USER`   |
|  8 | `STARRED_ITEMS`   | *(no table)*     |
|  9 | `BOOKMARKS`       | *(no table)*     |
| 10 | `SEARCH_MESSAGES` | `SEARCH_MESSAGE` |
| 11 | `SEARCH_FILES`    | `SEARCH_FILE`    |

**Note:** `STARRED_ITEMS` (8) and `BOOKMARKS` (9) are defined in the enum and
TYPES table but have no corresponding storage table and no assembler — they
would return an error if encountered in `insertPayload`. See `split.go`.

**Note:** `CHANNELS` (4) and `CHANNEL_INFO` (5) both write to the `CHANNEL`
table. `CHANNEL_INFO` (5) contains individually fetched full channel details;
`CHANNELS` (4) is the bulk list. `source.Channels()` prefers type 5, falling
back to type 4 if empty. See `source.go`.

---

## Key Data Quirks

### 1. No upsert — deduplication is at query time

All inserts use plain `INSERT INTO` with no `ON CONFLICT` clause
(`generic.go: stmtInsert`). The same message can appear in multiple `CHUNK`
rows (e.g. from pagination overlap or a `slackdump resume` run).

**Always select the row from the highest `CHUNK_ID` for a given message.**
The internal pattern used is a CTE:

```sql
WITH LATEST AS (
    SELECT CHANNEL_ID, MAX(CHUNK_ID) AS CHUNK_ID
    FROM MESSAGE
    GROUP BY CHANNEL_ID
)
SELECT M.*
FROM MESSAGE M
JOIN LATEST L ON M.CHANNEL_ID = L.CHANNEL_ID AND M.CHUNK_ID = L.CHUNK_ID
```

See `generic.go: stmtLatestRows`.

### 2. MESSAGE.ID is not a Slack timestamp string

`MESSAGE.ID` is the Slack timestamp (`TS`) converted to `int64` microseconds
by stripping the dot:

```
"1648085300.726649"  →  1648085300726649
```

This is done by `fasttime.TS2int()` (`fasttime/fasttime_x64.go`).

The primary key is `(ID, CHUNK_ID)`, so the same logical message appearing in
two chunks has the same `ID` but different `CHUNK_ID` values.

**Use `MESSAGE.TS` for the human-readable Slack timestamp.**

### 3. PARENT_ID is set on both parents and replies

`MESSAGE.PARENT_ID` is set to the `thread_ts` (as int64) for **all** messages
that have a thread timestamp — both the thread-parent and its replies.

- Thread-parent: `PARENT_ID = ID` (points to itself), `IS_PARENT = TRUE`
- Thread reply: `PARENT_ID = <parent's ID>`, `IS_PARENT = FALSE`

Filtering `PARENT_ID IS NOT NULL` alone matches both parents **and** replies.

| Goal | Filter |
|------|--------|
| Thread parents (with replies) | `IS_PARENT = TRUE` |
| Thread replies only | `IS_PARENT = FALSE AND PARENT_ID IS NOT NULL` |
| Non-threaded messages | `PARENT_ID IS NULL` |

### 4. IS_PARENT=TRUE implies the thread has replies

`IS_PARENT` is set by `structures.IsThreadStart()`:

```go
msg.ThreadTimestamp != "" &&
msg.Timestamp == msg.ThreadTimestamp &&
msg.LatestReply != "0000000000.000000"
```

A thread-lead message with `LatestReply = "0000000000.000000"` (deleted/empty
thread) gets `IS_PARENT = FALSE`. Therefore `IS_PARENT = TRUE` already
excludes empty threads — no need to additionally filter on `LATEST_REPLY`.

See `structures/conversation.go`.

### 5. Sentinel value "0000000000.000000"

Stored in `MESSAGE.LATEST_REPLY` and `LATEST_REPLY` column. Means the thread
was started but has no replies (deleted thread). Defined as
`structures.LatestReplyNoReplies`.

Do not attempt to fetch thread messages for rows with this value — there are
none.

### 6. thread_broadcast — messages duplicated across TYPE_ID=0 and TYPE_ID=1

Messages with `subtype = "thread_broadcast"` ("also sent to channel") are
stored in **both** the channel history chunk (TYPE_ID=0) and the thread chunk
(TYPE_ID=1).

When querying channel messages, filter them out to avoid double-counting:

```sql
WHERE JSON_EXTRACT(DATA, '$.subtype') IS NOT 'thread_broadcast'
```

See `repository/dbmessage.go: threadCond()`.

### 7. DATA column — full JSON blob

Every entity table stores the complete Slack API JSON payload in a `DATA`
column (`BLOB`, uncompressed). Explicit columns (`TS`, `PARENT_ID`,
`IS_PARENT`, `MODE`, `NAME`, etc.) are extracted for indexing purposes only.

All other fields — reactions, edited timestamps, message subtypes, user
profiles, blocks, attachments — are only accessible via `JSON_EXTRACT`:

```sql
SELECT TS,
       JSON_EXTRACT(DATA, '$.subtype')            AS subtype,
       JSON_EXTRACT(DATA, '$.edited.ts')          AS edited_ts,
       JSON_ARRAY_LENGTH(DATA, '$.reactions')     AS reaction_count
FROM MESSAGE
WHERE CHUNK_ID = 44;
```

Note: an abandoned `unused.go` file (`//go:build ignore`) contains gzip/flate
compression code that was never activated. Data is always stored uncompressed.

### 8. CHUNK.FINAL — archive completeness

`CHUNK.FINAL = TRUE` marks the last API pagination page for a given
channel/thread. If no `FINAL = TRUE` chunk exists for a channel, or the last
chunk has `FINAL = FALSE`, the archive is incomplete for that channel.

The index `CHUNK_I1 ON CHUNK (CHANNEL_ID, SESSION_ID, TYPE_ID, FINAL)` was
added in migration `20250809050908` to make completeness checks efficient.

### 9. SESSION.FINISHED — interrupted archives

`SESSION.FINISHED = TRUE` is set only when `DBP.Close()` completes
successfully. A session interrupted mid-run (crash, network failure) leaves
`FINISHED = FALSE`. Data from such a session may be incomplete.

See `repository/session.go: Finalise()` and `dbase.go: Close()`.

### 10. Resume creates a new SESSION

`slackdump resume` inserts a new `SESSION` row with `PAR_SESSION_ID` pointing
to the previous session. It does **not** update existing rows. Data from
multiple sessions coexists in the same tables, distinguished by the
`CHUNK_ID → SESSION_ID` chain.

The resume logic uses `OptInclusive(false)` (exclusive lower bound) so the
last known message is not re-fetched, plus a configurable lookback window
(default 7 days) to catch new replies on older messages.

See `cmd/slackdump/internal/resume/resume.go`.

### 11. S_USER — not USER

The users table is named `S_USER`, not `USER`. `USER` is a reserved word in
SQLite. Querying `.schema USER` will return nothing.

The `USERNAME` column is derived via `structures.Username()` which returns
`COALESCE(u.Name, u.ID)` — never empty.

### 12. FILE.MESSAGE_ID is NULL for canvas files

`FILE.MESSAGE_ID` is nullable. It is `NULL` for channel canvas files (Slack
Spaces), which are not attached to a specific message. `FILE.THREAD_ID` is
also nullable (only set when the file belongs to a thread message).

### 13. File modes

`FILE.MODE` comes directly from Slack's API. Known values:

| Mode | Downloadable | Notes |
|------|-------------|-------|
| `hosted` | Yes | Normal Slack-hosted file |
| `snippet` | Yes | Code snippet |
| `space` | Yes | Slack canvas / huddle space |
| `external` | No | Externally hosted; `is_external = true` in DATA |
| `tombstone` | No | File was deleted |
| `hidden_by_limit` | No | Hidden on free workspaces after 90 days |

See `convert/transform/fileproc/fileproc.go: invalidModes`.

### 14. CHUNK.NUM_REC is informational only

`CHUNK.NUM_REC` stores the record count at insert time from the chunk struct.
The actual number of `MESSAGE` rows for a given `CHUNK_ID` may differ. Do not
rely on `NUM_REC` for exact message counts — always `COUNT(*)` the target
table.

### 15. SEARCH_MESSAGE.ID is autoincrement

Unlike `MESSAGE` whose `ID` is derived from the Slack timestamp,
`SEARCH_MESSAGE.ID` is a SQLite autoincrement integer. Search results have no
deduplication key equivalent to `MESSAGE.TS`; the latest-chunk pattern
(MAX CHUNK_ID per CHANNEL_ID) is used instead.

### 16. Database pragmas

The database is initialised with:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous  = NORMAL;
PRAGMA foreign_keys = ON;
```

External tools querying the database should be aware of WAL mode (the
`slackdump.sqlite-wal` and `slackdump.sqlite-shm` sidecar files must be
present for a consistent read while slackdump is running).

See `dbase.go: dbInitCommands`.

---

## Relevant Source Files

| File | What it covers |
|------|---------------|
| `repository/migrations/*.sql` | Full schema and all views |
| `repository/generic.go` | Insert pattern (no upsert), latest-chunk CTE |
| `repository/dbmessage.go` | `DBMessage`, `threadCond()`, `IS_PARENT` logic, `JSON_EXTRACT` usage |
| `repository/dbfile.go` | `DBFile`, file mode stored directly |
| `repository/dbchannel.go` | `DBChannel`, channel type priority (INFO vs bulk) |
| `repository/dbuser.go` | `DBUser`, `S_USER` table, `Username()` |
| `repository/session.go` | `Session`, `PAR_SESSION_ID`, `Finalise()` |
| `repository/unused.go` | Abandoned gzip compression (`//go:build ignore`) |
| `dbase.go` | `DBP`, DB init pragmas, `Close()` finalises session |
| `split.go` | `InsertChunk()`, `insertPayload()` dispatch |
| `source.go` | `Source`, `Channels()` fallback, `Latest()` for resume |
| `assemble.go` | Chunk reassembly, parent message lookup |
| `../chunk.go` | `ChunkType` enum (0–11) |
| `../../structures/conversation.go` | `IsThreadStart()`, `LatestReplyNoReplies` sentinel |
| `../../fasttime/fasttime_x64.go` | `TS2int()` — timestamp → int64 encoding |
| `../../../../convert/transform/fileproc/fileproc.go` | `invalidModes` |
| `../../../../cmd/slackdump/internal/resume/resume.go` | Resume session logic |
