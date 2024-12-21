# Slackdump Channel List Syntax

Slackdump major modes like `archive`, `export` and `dump` support
including or excluding channels from the operation. This document
describes how to use the inclusive and exclusive modes, along with
examples.

Slackdump accepts channel IDs or URLs as arguments separated by space.
The channel ID is the last part of the channel URL. For example, in the
URL

```
https://xxx.slack.com/archives/C12345678
```

the channel ID is `C12345678`.

You can also get all available channel IDs by running the `slackdump list channels` command.

## Syntax

- No prefix: include the channel in the operation.
- `^`: exclude the channel from the operation.
- `@`: read the channels from a file.

File can contain one or more channel IDs or URLs, one per line.

Below, we'll look at some examples.

## Examples

### Exporting Only Channels You Need

To include only those channels you're interested in, use the following
syntax:

```bash
slackdump export C12401724 https://xxx.slack.com/archives/C4812934
```

The command above will export ONLY channels `C12401724` and `C4812934`.

### Exporting Everything Except Some Unwanted Channels

To exclude one or more channels from the export, prefix the channel with
the caret "^" character. For example, you want to export everything
except channel `C123456`:

```bash
slackdump -export my-workspace.zip ^C123456
```

### Providing the List in a File

You can specify the filename instead of listing all the channels on the
command line. To include the channels from the file, use the "@"
character prefix. The following example shows how to load the channels
from the file named "data.txt":
```bash
slackdump archive @data.txt
```
It is also possible to combine files and channels, i.e.:
```bash
slackdump archive @data.txt ^C123456
```
The command above will read the channels from data.txt and exclude the
channel `C123456` from the Export.

