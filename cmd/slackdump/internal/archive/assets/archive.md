# Archive Command

The archive command archives the Slack workspace into a directory of files.
By default, it will perform the archiving of the full workspace that is accessible
to your user.

Optionally, one can select channels, groups and DMs to archive.  For this, one
needs to specify the URLs or IDs of the channels.

The workspace is archived in the "Chunk" format, that can be viewed with the
`view` command, or converted to other supported formats, such as native Slack
Export format.

## What is contained in the archive?

"Archive" behaves similarly to the Slackdump export feature, the output of a
successful run contains the following:
- channels.json.gz - list of channels in the workspace (full archives only);
- users.json.gz - list of users in the workspace;
- CXXXXXXX.json.gz - channel or group conversation messages, where XXXXXXX is
  the channel ID;
- DXXXXXXX.json.gz - direct messages, where XXXXXXX is the user ID;

Output format:

- Each file is a JSONL file compressed with GZIP.

Please note that "archive" can not create ZIP files, but you can zip the output
directory manually.

## What is chunk file format?

Run `slackdump help chunk` for the format specification.
