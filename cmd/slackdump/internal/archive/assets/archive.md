# Archive Command

The `archive` command saves your Slack workspace as a directory of files. By
default, it archives the entire workspace that your user can access. You can
customize the archive to include specific channels, groups, or direct messages
by providing their URLs or IDs.

The benefits of using "Archive" over "Export" and "Dump" are:
- it is well documented (see `slackdump help chunk`);
- it can be converted to other formats, including the native Slack Export
  format (see `slackdump help convert`);
- it is easier to parse with tools like `jq` or `grep`;
- it is more convenient to build your own tools around it;
- it is used internally by Slackdump to generate Slack Export and dump files,
  so it can be seen as "master" format for the data;
- It is natively supported by `slackdump view` command, everything else uses
  an adapter.

## Features

### Default Behaviour
- Archives the full workspace accessible to your user.

### Optional Customization
- Specify channels, groups, or DMs to archive by providing their URLs or IDs.

### Output Format
- The archive uses the **"Chunk" format**, which can be:
  - Viewed using the `view` command.
  - Converted to other formats, including the native Slack Export format.

## Archive Contents

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

## What is the Chunk Format?

The Chunk format is a specific structure used for archiving data. For details
on this format, run:  `slackdump help chunk`

