# Command Record

The record command runs the complete dump of the workspace, if no channels or
threads are given on the command line or in the file.  The recording is in the
"Chunk" file format.

## What does Record dump?

Record behaves similarly to the Slackdump export feature, the output of a
successful run contains the following:
- channels.json.gz - list of channels in the workspace;
- users.json.gz - list of users in the workspace;
- CXXXXXXX.json.gz - channel or group conversation messages, where XXXXXXX is
  the channel ID;
- DXXXXXXX.json.gz - direct messages, where XXXXXXX is the user ID;

Output format:

- The output is saved into JSONL files (each line is a JSON object).
- The output is a directory with GZIP-compressed files, "record" can not write
  to ZIP archives.

## Chunk file format

Run `slackdump help chunk` for the format specification.
