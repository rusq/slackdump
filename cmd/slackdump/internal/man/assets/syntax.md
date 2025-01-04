# Slackdump Channel List Syntax

Slackdump major commands like `archive`, `export`, and `dump` allow
you to include or exclude specific channels from an operation.  This
document explains the inclusive and exclusive modes, their syntax,
and provides examples for practical use.

## Executive Summary

- Entities are separated by spaces on the CLI, and by new lines in files.
- __FILES__: Use the `@` prefix for files, example: `@channels.txt`.
  - each line in the file should contain a channel or thread ID or URL of a
    channel or thread, or a comment if the line starts with `#`.
  - it supports the same syntax as the command line for time ranges and
    exclusion.
- __EXCLUSION__: Use the `^` prefix for exclusions, example: `^C123`.
- __TIME RANGE__: Time range parameters are optional but can refine your export
  or archive operation.
  - Time ranges are specified as `time_from` and `time_to` in
    `YYYY-MM-DDTHH:MM:SS` format.
  - They should be separated by commas without spaces.

## Syntax

Slackdump accepts channel IDs or URLs as arguments, separated by
spaces, if specified on the command line, and by new line characters,
if supplied in a file.  URLs and IDs can be used interchangeably:

Here's the Slack Channel URL, where the **channel ID** is the last part of the
URL:

```
https://xxx.slack.com/archives/C051D4052
```

the channel ID is `C051D4052`.

To get a list of all available channel IDs, run:
```bash
slackdump list channels
```

The syntax for specifying entities is as follows:
```
[[prefix]term[,[time_from],[time_to]]|@file]
```

Please note that there are no spaces before or after the commas ",".

Where:
- `prefix`: Determines how the channel is processed.
  - No prefix: Include the channel in the operation.
  - `^`: Exclude the channel from the operation.
- `term`: can be one of the following:
  - Channel ID (i.e. C051D4052)
  - Thread ID i.e. "C051D4052:1665917454.731419",
  - URL of the channel or thread(see above)
- `time_from` and `time_to`: Optional parameters specifying the time range for
  the operation in `YYYY-MM-DDTHH:MM:SS` format.
  - If only `time_from` is specified, the operation includes all messages
    starting from that time.
  - If only `time_to` is specified, the operation includes all messages up to
    that time.
  - If both are specified, the operation includes messages within that time
    range.
- `@file`: A file containing channel or thread IDs or URLs:
  - each entry on a new line;
  - comments start with `#`, and should be on a new line;
  - empty lines are skipped.

## Examples

### 1. Exporting Specific Channels or Threads

To include only specific channels in the operation:

```bash
slackdump export C12401724 https://xxx.slack.com/archives/C4812934
```

This command exports **only** channels `C12401724` and `C4812934`.

To include specific threads you can provide the thread URL:
```bash
slackdump dump \
    https://ora600.slack.com/archives/C051D4052/p1665917454731419
```
or use the Slackdump thread notation:
```bash
slackdump export C051D4052:1665917454.731419
```

### 2. Exclude Specific Channels or Threads

To exclude one or more channels, prefix them with ^.  For example,
to export everything except channel C123456:

```bash
slackdump export ^C123456
```
This excludes `C123456` while exporting the rest.

### 3. Using a File for Channel Lists

You can specify a file containing channel IDs or URLs.  The file should contain
IDs, one per line. To include channels from a file:

```bash
slackdump archive @data.txt
```
You can also combine files and individual channel exclusions.  For
example:

```bash
slackdump archive @data.txt ^C123456
```
This command includes channels listed in data.txt but excludes C123456.

Sample file:
```text
# This is a comment
C123456
^C123457
https://ora600.slack.com/archives/C051D4052/p1665917454731419
```

### 4. Using Time Ranges

To include messages from a specific time range:

```bash
slackdump archive C123456,2022-01-01T00:00:00,2022-01-31T23:59:59
```

This command archives messages from channel `C123456` between January
1st and January 31st, 2022.

Before some date:

```bash
slackdump archive C123456,2022-01-01T00:00:00
# or
slackdump archive C123456,2022-01-01T00:00:00,
```

After some date:

```bash
slackdump archive C123456,,2022-01-31T23:59:59
```

