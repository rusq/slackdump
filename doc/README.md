# Slackdump User Guide

## Table of Contents

- [Installation](#installation)
- [Logging In](#logging-in)
  - [Automatic (browser-based) login](login-automatic.md)
  - [Manual login (token/cookie)](login-manual.md)
- [Usage](#usage)
  - [Archiving a workspace](usage-archive.md)
  - [Listing users/channels](usage-list.md)
  - [Dumping messages and threads](usage-channels.md)
  - [Creating a Slack Export](usage-export.md)
  - [Downloading all Emojis](usage-emoji.md)
- [Compiling from Sources](compiling.md)
- [Troubleshooting](troubleshooting.md)

## Installation

Installing is simple — download the latest Slackdump from the
[Releases](https://github.com/rusq/slackdump/releases) page, extract and run it:

1. Download the archive from the [Releases](https://github.com/rusq/slackdump/releases)
   page for your operating system.

   > **macOS users** can use `brew install slackdump` to install the latest version.

2. Unpack the archive.
3. Change directory to where you unpacked it.
4. Run `./slackdump` (or `slackdump.exe` on Windows) to start the wizard.

For compiling from sources, see [Compiling from Sources](compiling.md).

## Logging In

See [Automatic Login](login-automatic.md) for the recommended browser-based
login methods (Interactive, User Browser, Headless, and QR Code / Sign in on
Mobile).

For manual token/cookie authentication (headless/CI environments), see
[Manual Authentication](login-manual.md).

To import a saved token/cookie file:

```shell
slackdump workspace import <filename>
```

## Usage

There are several modes of operation:

| Command | Description |
|---------|-------------|
| `slackdump archive` | Archive the whole workspace (or specific channels) to a SQLite database |
| `slackdump export` | Export to Slack-compatible ZIP (Standard or Mattermost format) |
| `slackdump dump` | Dump individual conversations or threads to JSON |
| `slackdump list users` | List all workspace users |
| `slackdump list channels` | List all visible channels |
| `slackdump emoji` | Download all workspace custom emojis |
| `slackdump resume` | Resume a previously interrupted archive |
| `slackdump convert` | Convert an archive to another format |
| `slackdump view` | View a dump, export or archive in the browser |
| `slackdump search` | Dump Slack search results |
| `slackdump mcp` | Start a local MCP server for AI agent access |

Run `slackdump help` to see all available commands, or `slackdump help <command>`
for detailed help on a specific command.

## Beginner's Guide to the Command Line

If you have no experience with the Linux/macOS Terminal or Windows Command
Prompt, the [Unix Shell Guide](https://swcarpentry.github.io/shell-novice/)
is a good starting point.
