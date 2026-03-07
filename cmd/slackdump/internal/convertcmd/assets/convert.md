# Convert Command

Converts between different Slackdump supported formats.

## Usage
```bash
slackdump convert [-f format] [-o output] <input>
```

Where format is one of the following:
- `chunk` **Chunk format**: JSON.GZ files with metadata, output is a directory.
- `database` **SQLite database**: SQLite database format used by Slackdump, output is a directory.
- `dump` **Dump**: JSON files where each channel is a large JSON object. Output is a directory or a zip file.
- `export`: **Slack Export**: The native Slack export format. Output is a directory or a zip file.

By default Slackdump converts to Slack Export format and writes to a ZIP file
output.

If any files were saved in the source location, they will be copied to the target directory or ZIP file, unless
`-files=false` is specified.

To copy avatars, use `-avatars` flag.  By default, avatars are not copied.

## Example

Convert Slack Export to database format:
```bash
slackdump convert -f database -o MyArchive/ slack_export.zip
```

Converting from database format to Slack Export format:
```bash
slackdump convert -f export -o my_archive.zip slackdump_20211231_150405/
```
Note, that there's no necessity to specify the "slackdump.sqlite" file, Slackdump
will automatically find it in the directory.

See also:
- `slackdump help archive`
