# Slackdump Channel List Syntax

Slackdump major commands like `archive`, `export`, and `dump` allow you
to include or exclude specific channels from an operation. This document
explains the inclusive and exclusive modes, their syntax, and provides
examples for practical use.

## Syntax

Slackdump accepts channel IDs or URLs as arguments, separated by spaces.  
The **channel ID** is the last part of the channel URL. For example, in the URL:

```
https://xxx.slack.com/archives/C12345678
```

the channel ID is `C12345678`.

To get a list of all available channel IDs, run:
```bash
slackdump list channels
```

The syntax for specifying entities is as follows:
```
[[prefix]term[/[time_from]/[time_to]]|@file]
```

Where:
- `prefix`: Determines how the channel is processed.
  - No prefix: Include the channel in the operation.
  - `^`: Exclude the channel from the operation.
- `term`: The channel ID, URL, or filename.
- `time_from` and `time_to`: Optional parameters specifying the time
  range for the operation in `YYYY-MM-DDTHH:MM:SS` format.
  - If only `time_from` is specified, the operation includes all messages
    starting from that time.
  - If only `time_to` is specified, the operation includes all messages
    up to that time.
  - If both are specified, the operation includes messages within that
    time range.

A file can contain one or more channel IDs or URLs, with each entry on a
new line.

## Examples

### 1. Exporting Specific Channels

To include only specific channels in the operation:

```bash
slackdump export C12401724 https://xxx.slack.com/archives/C4812934
```

This command exports **only** channels `C12401724` and `C4812934`.

### 2. Exclude Specific Channels

To exclude one or more channels, prefix them with ^. For example, to
export everything except channel C123456:

```bash
slackdump export ^C123456
```
This excludes `C123456` while exporting the rest.

### 3. Using a File for Channel Lists

You can specify a file containing channel IDs or URLs. To include
channels from a file:

```bash
slackdump archive @data.txt
```
You can also combine files and individual channel exclusions. For
example:

```bash
slackdump archive @data.txt ^C123456
```
This command includes channels listed in data.txt but excludes C123456.

### 4. Using Time Ranges

To include messages from a specific time range:

```bash
slackdump archive C123456/2022-01-01T00:00:00/2022-01-31T23:59:59
```

This command archives messages from channel `C123456` between January 1st
and January 31st, 2022.

## TL;DR

- Use the `@` prefix for files and the `^` prefix for exclusions.
- Time range parameters are optional but can refine your export or
  archive operation.

