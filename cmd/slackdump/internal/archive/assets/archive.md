# Archive Command

The `archive` command saves your Slack workspace as a SQLite database. By
default, it archives the entire workspace that your user can access. You can
customize the archive to include specific channels, groups, or direct messages
by providing their URLs or IDs on the command line or in the Wizard.

The database is located in `slackdump.sqlite` in the output directory.

Alternatively, you can use `-legacy` flag to archive into chunk file format, if
you experience problems with the database.  Note, that `-legacy` flag is
temporary for the transition period, and will be removed in v3.2.0.
<!-- TODO: remove above paragraph once -legacy is deleted -->

The benefits of using "Archive" over "Export" and "Dump" are:
- it is faster, because it does not need to convert the data to export or dump
  formats;
- the database can be easily queried using SQL using SQLite CLI or SQLite
  Browser;
- the archiving can be "resumed" by `resume` command, meaning that you can
  continue where you left off in case of previous failure or incremental
  backups.
- it can be converted to other formats, including the native Slack Export
  format (see `slackdump help convert`);
- it is more convenient to build your own tools around it;
- it is used internally by Slackdump to generate Slack Export and dump files,
  so it can be seen as "master" format for the data;

## Features

## Database Archive Contents

The archive contains the following files:

- **`slackdump.sqlite`**: The SQLite database file containing all the data
  from the workspace.
- **`__uploads`**: A directory containing files attached to messages that were
  downloaded, if the file download is enabled.
- **`__avatars`**: A directory containing user avatars that were downloaded,
  if the avatar download is enabled.

Sometimes you might see `slackdump.sqlite-shm` and `slackdump.sqlite-wal` files
in the output directory. These are temporary files created by SQLite for
performance reasons. They are not necessary for the archive, unless Slackdump
was interrupted or crashed. You can safely delete them if you are sure that the
archive is complete.

### Database Structure

The database contains the following tables:
- **CHANNEL**:  Contains all channels in the workspace.
- **CHANNEL_USER**:  Contains the mapping between channels and users.
- **CHUNK**:  Contains the "chunk" metadata, including the chunk type, number
  of records retrieved and the SESSION ID.
- **FILE**:  Contains all discovered file metadata from messages.
- **MESSAGE**:  Contains all messages and thread messages from the workspace.
- **SEARCH_FILE**:  Contains search results for files.
- **SEARCH_MESSAGE**:  Contains search results for messages.
- **SESSION**:  Contains the session information, including the start and end
  time of the period.
- **S_USER**:  Contains all users in the workspace.
- **WORKSPACE**: Contains the workspace information, including the workspace ID
  and name.

There are also additional views, starting with `V_`, which are used by Slackdump
during the archiving process, they should not be removed or modified.

## Legacy Archive Contents

The archive behaves like the Slackdump export feature. A successful run
output includes:

- **`channels.json.gz`**: A list of channels in the workspace (only for full
  archives).
- **`users.json.gz`**: A list of users in the workspace.
- **`CXXXXXXX.json.gz`**: Messages from a channel or group, where `XXXXXXX` is
  the channel ID.
- **`DXXXXXXX.json.gz`**: Direct messages, where `XXXXXXX` is the user ID.
- **`__uploads/`**: A directory containing files attached to messages that were
  downloaded, if the file download is enabled.

### File Format
- Files are saved as **JSONL** (newline-delimited JSON) and compressed with
  **GZIP**.
- Note: The `archive` command does not create ZIP files, but you can manually
  compress the output directory into a ZIP file if needed.

For details on this format, run:  `slackdump help chunk`

## Migrating from v3.x
If you're using Chunk files in your tooling, you can convert the database to the
chunk format using the `convert` command. For example:
```bash
slackdump convert -f chunk ./slackdump_20211231_123456
```

or

```bash
slackdump convert -f chunk ./slackdump_20211231_123456/slackdump.sqlite
```
