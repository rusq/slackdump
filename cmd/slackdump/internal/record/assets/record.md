# Command Record

The record command runs the complete dump of the workspace.  The dump is in
the "Chunk" file format.

## What does Record dump?

Record behaves similarly to the Slack export feature, the output of a
successful run contains the following:
- channels.json.gz - list of channels in the workspace;
- users.json.gz - list of users in the workspace;
- CXXXXXXX.json.gz - channel or group conversation messages, where XXXXXXX is
  the channel ID;
- DXXXXXXX.json.gz - direct messages, where XXXXXXX is the user ID;

Please note that these are not traditional JSON files, but rather JSONL files,
where each line is a JSON object.  This is done to minimise the memory usage
for processing.

Another difference to the Slack export is that the output is not a single
archive, but rather a directory with files.  Slackdump does not support writing
chunk files into a ZIP file, and strictly speaking, it is not necessary, as 
chunk files are already compressed.

## Chunk file format

See `slackdump help chunk` for the format specification.
