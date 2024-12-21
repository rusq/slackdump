# Command: archive

The `archive` command saves your Slack workspace as a directory of files. By
default, it archives the entire workspace that your user can access. You can
customize the archive to include specific channels, groups, or direct messages
by providing their URLs or IDs.

## Features

### Default Behavior
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

### File Format
- Files are saved as **JSONL** (newline-delimited JSON) and compressed with
  **GZIP**.
- Note: The `archive` command does not create ZIP files, but you can manually
  compress the output directory into a ZIP file if needed.

## What is the Chunk Format?

The Chunk format is a specific structure used for archiving data. For details
on this format, run:  `slackdump help chunk`

