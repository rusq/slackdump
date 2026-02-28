---
name: slackdump
description: Collect the Slack conversation data from Slackdump Archive format.
compatibility: opencode
metadata:
  audience: general
  workflow: archive
---

## Accessing the Slackdump Data

Flowchart:

```
Slackdump MCP available?
  yes → USE IT (stop here)
  no  ↓
Source has slackdump.sqlite?
  no  → REFUSE; tell user: slackdump convert -f database <source>
  yes ↓
SQLite MCP available?
  yes → USE IT (stop here)
  no  ↓
sqlite3 CLI available?
  yes → use sqlite3 CLI (slackdump-sqlite3 skill)
  no  → REFUSE to run
```


Use the "Slackdump" MCP if it is available. It is read-only and supports all
type of Slackdump archives.

If you need to identify the source type, use the following guidance:
- YYYYMMDD and HHMMSS represent the 24-hour format time when the slackdump was
  invoked to generate this archive (it is created as a first step);
- Source types:
  1. A directory with database: identified by `slackdump_[YYYYMMDD]_[HHMMSS]/slackdump.sqlite`
  2. A standalone database: identified by `slackdump.sqlite`
  3. A chunk directory: identified by `slackdump_[YYYYMMDD]_[HHMMSS]/*.json.gz`
  4. A dump directory: identified by `slackdump_[YYYYMMDD]_[HHMMSS]/[CHANNEL_ID].json`
  5. An export directory: identified by `slackdump_[YYYYMMDD]_[HHMMSS]/dms.json`
  6. An export or dump zip file: identified by
     `slackdump_[YYYYMMDD]_[HHMMSS].zip`. In the ZIP file, the file structure is
     similar to dump and export directories described above.

- If you need to understand the source structure in details, use
  "slackdump-source" skill.

IMPORTANT: You will be accessing the data in read-only mode. You must not
UPDATE or DELETE or run any DML or DDL statements. Only SELECT and
data-dictionary querying.

- Database path might be given by the agent, otherwise look for
  `slackdump.sqlite` file. If there are several files like this in the current
  project, ask user to choose one of them.

- If falling back to `sqlite3` use the "slackdump-sqlite3" skill.

## What to pay attention to

This section describes some data structure quirks that you need to be aware of
in order to understand what you're dealing with.

### Threads
- Threads are nested conversations. Here's how a thread looks like:
  ```
  +- parent_message (ts=12345, thread_ts=12345, latest_reply=XXXX)
  | +-- parent_message (ts=12345, thread_ts=12345)    - conversations.replies endpoint returns the starter message, so it MIGHT be duplicated in the archive
  | +-- first_thread_message(ts=12346, thread_ts=12345)
  | +-- second_thread_message(ts=12355, thread_ts=12345)
  | +-- <...>
  | +-- n'th_thread_message(ts=XXXX, thread_ts=12345)
  +- next_non-threaded_message_in_the_channel (ts=24921, thread_ts=<empty>)
  ```
- Note that the parent_message appears twice. One is returned during the
  `conversations.history,` and a duplicate is returned by
  `conversation.replies` endpoints. It might be duplicated in the archive.

- In the channel there may be parent messages with deleted threads,
  `latest_reply` field will be set to a special zero-value "0000000000.000000",
  there will be no thread messages for such threads, so no need to call
  `get_thread` tool on MCP:
  ```
  +- parent_message (ts=23456, thread_ts=23456, latest_reply=0000000000.000000)
  |  // there will be no thread messages
  +- next_message (can be threaded or non threaded)
  ```

- `IS_PARENT=TRUE` already implies the thread has replies — a thread-lead
  message with `latest_reply = "0000000000.000000"` is stored with
  `IS_PARENT=FALSE`. You do not need to additionally filter by `latest_reply`
  when looking for threads that have replies.

- `IS_PARENT=FALSE` indicates a thread message that is not a parent message of
  a thread (thread reply message), or a parent message of a deleted thread.

- Messages with subtype `thread_broadcast` ("also sent to channel") appear in
  **both** the channel history (CHUNK.TYPE_ID=0) and the thread
  (CHUNK.TYPE_ID=1). When counting or listing channel messages, filter them out
  to avoid double-counting:
  ```sql
  WHERE JSON_EXTRACT(DATA, '$.subtype') IS NOT 'thread_broadcast'
  ```

### Files
Some messages may have non empty files array. A file has a "mode" field. Known
file modes:

- `hosted`: normal Slack-hosted file, downloadable;
- `snippet`: code snippet hosted by Slack;
- `space`: Slack canvas / huddle space document;
- `external`: not hosted on Slack servers, `is_external` will be `true`, not downloadable;
- `tombstone`: file was deleted, `download_url` will be empty;
- `hidden_by_limit`: on free workspaces, Slack hides files uploaded 90+ days ago, `download_url` will be empty.

Only `hosted`, `snippet`, and `space` files are downloadable. The others (`external`,
`tombstone`, `hidden_by_limit`) have no usable download URL.

NOTE: do not try to download, do the following:
- If the directory with `slackdump.sqlite` has `/__uploads` subdirectory,
  find it using the following pattern
  `/__uploads/[FILE_ID]/*`.
- Otherwise: files are not downloaded; WARN user that files are not present,
  and user can use `slackdump tools redownload`, if files are needed.

## If you're accessing database directly

Examine the database schema to understand the structure of the data:
- Use Foreign Keys to `JOIN` tables.
- Ignore `V_*` views (context: they are used by slackdump to track unprocessed
  threads during execution).

Key terms:

- Session: stored in a `SESSION` table, and denotes a single `slackdump`
  execution. Each `slackdump` invocation on the archive creates a session. The
  following commands create a session entry:

  - `slackdump archive` - archival of data
  - `slackdump resume` - incremental archiving of data (adds to existing
    archive)
  - `slackdump search` - writes search results
  - `slackdump convert` - creates a session to store the data from the source
    format.

  A session interrupted mid-run (e.g. crash or network failure) will have
  `FINISHED=FALSE`. Data from such a session may be incomplete.

- Chunk:  stored in a `CHUNK` table.  A "chunk" loosely maps to a single API
  call made by Slackdump when making an archive. When Slack endpoint returns
  some data, Slackdump creates a "chunk" entry, and then inserts the data into
  one of the linked database tables.  For example: a call to Slack API endpoint
  "conversations.history" returns 100 messages.  For this API call, a new CHUNK
  is inserted with `TYPE_ID` set to "MESSAGES" type. Then it will insert 100
  rows into `MESSAGE` table. Each of these rows will be linked to this chunk.

  A chunk with `FINAL=TRUE` is the last API page for that channel or thread. If
  the last chunk for a channel has `FINAL=FALSE`, or no `FINAL=TRUE` chunk
  exists at all for it, the archive is incomplete for that channel.

- Chunk Type:  stored in a `TYPES` table.  Chunk type is helpful to understand
  what type of API call was made and which table in database contains the data
  for that "chunk".

The same message (having the same MESSAGE.TS) can appear multiple times in different chunks and different sessions.
You always need to fetch the latest version of the message (the one that belongs to the latest chunk, and the latest session).

Example:

Here's the flattened representation of the result that might be returned by:
```sql
SELECT SESSION.ID, CHUNK.ID, MESSAGE.TS
FROM MESSAGE
JOIN CHUNK ON CHUNK.ID = MESSAGE.CHUNK_ID
JOIN SESSION ON SESSION.ID = CHUNK.SESSION_ID
WHERE MESSAGE.TS IN ('12345.678', '12345.890');
```

| SESSION.ID | CHUNK.ID | MESSAGE.TS |
| ---------- | -------- | ---------- |
|   1        |    44    | 12345.678  |
|   1        |    44    | 12345.890  |
|   2        |    104   | 12345.678  |

In this case, message 12345.678 appears twice, first time in the first session,
second time in the second session (which, most likely, a `slackdump resume`
session).  You should pick the latest version of the message (`SESSION.ID =
2, CHUNK.ID=104`).


