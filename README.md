# Slack Dumper

Purpose:  archive your private and public Slack messages, users, channels,
files and emojis.  Generate Slack Export without admin privileges.

[![Slackdump screenshot](doc/slackdump.webp)](https://github.com/rusq/slackdump/releases/)

**Quick links**:

- [Installation And Quickstart](#installation-and-quickstart)
- [Join the discussion in Telegram](https://t.me/slackdump).
- [Buy me a cup of tea](https://ko-fi.com/rusq_), or use **GitHub Sponsors**
  button on the top of the page.
- [![Go reference](https://pkg.go.dev/badge/github.com/rusq/slackdump/v4.svg)][godoc]
- [Using with AI Agents (MCP Server)](#slackdump-mcp-server)
- [Slack MCP Server (no permissions required)](https://github.com/korotovsky/slack-mcp-server)
- [Github CLI Slackdump extension](https://github.com/wham/gh-slackdump)
- How to's:

  - [Mattermost migration][mmost] steps
  - [SlackLogViewerとSlackdumpを一緒に使用する](https://kenkyu-note.hatenablog.com/entry/2022/09/02/090949)
  - [v1 Overview on Medium.com](https://medium.com/@gilyazov/downloading-your-private-slack-conversations-52e50428b3c2)  (outdated)

[godoc]: https://pkg.go.dev/github.com/rusq/slackdump/v4/
[mmost]: doc/usage-export.md
[ug]: doc/README.md


> [!WARNING]
> # Enterprise Workspaces Security Alerts
>
> Depending on your Slack plan and security settings, using Slackdump may
> trigger Slack security alerts and/or notify workspace administrators of
> unusual or automated access/scraping attempts.
> 
> You are responsible for ensuring your use complies with your organisation’s
> policies and Slack’s terms of service.
>
> **See [Enterprise Workspace Tips](doc/enterprise.md).**

## Description

Typical use scenarios:

* archive your private conversations from Slack when the administrator
  does not allow you to install applications OR you don't want to use
  potentially privacy-violating third-party tools;
* archive channels from Slack when you're on a free "no archive" subscription;
  so you don't lose valuable knowledge in those channels;
* create a Slack Export archive without admin access;
* create incremental Slack archives, which is particularly useful for free
  workspaces with 90-day limits;
* save your favourite emojis; AND
* analyse you Slack data with AI Agent using [Slackdump MCP Server](#slackdump-mcp-server).

There are several modes of operation

1. List users/channels
1. Dumping messages and threads
1. Creating a Slack Export in Mattermost or Standard modes.
1. Creating an Archive (Sqlite database or stored as json+gz)
1. Converting an archive to other formats (Export, Dump).
1. Emoji download mode.
1. Viewing Slack export, dump or archive files or directories (displays images).
1. Search mode (messages and files).
1. Resuming previous archive (adding new messages to an existing archive).
1. Local MCP Server to use with Opencode, Claude, or any other AI tool
   supporting mcp over STDIO or HTTP.

Run `slackdump help` to see all available options.

## Installation and Quickstart

On macOS, you can install Slackdump with Homebrew:

```shell
brew install slackdump
```

On other Operating Systems, please follow these steps:

1. Download the latest release for your operating system from the [releases] page.
1. Unpack the archive to any directory.
1. Run the `./slackdump` or `slackdump.exe` executable (see note below).
1. You know the drill:  use arrow keys to select the menu item, and Enter (or
   Return) to confirm.
1. Follow these [quickstart instructions][man-quickstart].

[releases]: https://github.com/rusq/slackdump/releases/

> [!NOTE] 
> On Windows and macOS you may be presented with "Unknown developer" window,
> this is fine.  Reason for this is that the executable hasn't been signed by
> the developer certificate.

  To work around this:

  - **on Windows**: click "more information", and press "Run
    Anyway" button.
  - **on macOS** 14 Sonoma and prior:  open the folder in Finder, hold Option
    and double click the executable, choose Run.
  - **on macOS** 15 Sequoia and later:  start the slackdump, OS will show the
    "Unknown developer" window, then go to System Preferences -> Security and
    Privacy -> General, and press "Open Anyway" button.

### Getting Help

- Quickstart guide: `slackdump help quickstart`, read [online][man-quickstart].
- Generic command overview: `man ./slackdump.1`
- [Ez-Login 3000](https://github.com/rusq/slackdump/wiki/EZ-Login-3000) Guide.
- What's new in V4: `slackdump help whatsnew`, read [online][man-changelog].

[man-quickstart]: https://github.com/rusq/slackdump/blob/master/cmd/slackdump/internal/man/assets/quickstart.md
[man-changelog]: https://github.com/rusq/slackdump/blob/master/cmd/slackdump/internal/man/assets/changelog.md

## Running Slackdump from a Repo Checkout

If you've cloned the repository and want to run slackdump directly without downloading a release, you can do one of the following:

1. **Build and run** (creates an executable):
   ```shell
   go build -o slackdump ./cmd/slackdump
   ./slackdump wiz
   ```

2. **Run directly**:
   ```shell
   go run ./cmd/slackdump wiz
   ```

Note: You need Go installed on your system (see `go.mod` for the version)


## Slackord2: Migrating to Discord

If you're migrating to Discord, the recommended way is to use
[Slackord2](https://github.com/thomasloupe/Slackord2) — a great tool with a
nice GUI, that is compatible with the export files generated by Slackdump.

## User Guide

For more advanced features and instructions, please see the [User Guide][ug],
and read `slackdump help` pages.

# Previewing Results

Once the workspace data is dumped, you can run built-in viewer:

```shell
slackdump view <zip or directory>
```

The built-in viewer supports all types of dumps:

1. Slackdump Archive format;
1. Standard and Mattermost Slack Export;
1. Dump mode files
  
The built-in viewer is experimental, any contributions to make it better looking are welcome.

Alternatively, you can use one of the following tools to preview the
export results:

- [SlackLogViewer] - a fast and powerful Slack Export viewer written in C++, works on Export files (images won't be displayed, unless you used an export token flag).
- [Slackdump2Html] - a great Python application that converts Slack Dump to a
  static browsable HTML.  It works on Dump mode files.
- [slack export viewer][slack-export-viewer] - Slack Export Viewer is a well known viewer for
  slack export files. Supports displaying files if saved in the "Standard" file mode.

[SlackLogViewer]: https://github.com/thayakawa-gh/SlackLogViewer/releases
[Slackdump2Html]: https://github.com/kununu/slackdump2html
[slack-export-viewer]: https://github.com/hfaran/slack-export-viewer

# Slackdump MCP server

Slackdump offers a read-only MCP server with the following features:
- analyse the data in the archive (any type)
- provide help with command line flags

Available MCP tools:

| Tool | Description |
|------|-------------|
| `load_source` | Open (or switch to) a Slackdump archive at runtime |
| `list_channels` | List all channels in the archive |
| `get_channel` | Get detailed info for a channel by ID |
| `list_users` | List all users/members |
| `get_messages` | Read messages from a channel (paginated) |
| `get_thread` | Read all replies in a thread |
| `get_workspace_info` | Workspace/team metadata |
| `command_help` | Get CLI flag help for any slackdump subcommand |

The server supports both **stdio** (agent-managed) and **HTTP** transports.

### Quick project setup

Scaffold a ready-to-use project directory pre-configured for your AI tool:

```shell
slackdump mcp -new opencode   ~/my-slack-project   # OpenCode
slackdump mcp -new claude-code ~/my-slack-project  # Claude Code
slackdump mcp -new copilot    ~/my-slack-project   # VS Code / GitHub Copilot
```

Each command creates the MCP config file and installs bundled Slackdump skill /
instruction files so the agent knows how to work with your archive out of the box.

To learn how to set it up with Claude Desktop, VS Code/GitHub Copilot, or
OpenCode, see:
```
slackdump help mcp
```
or refer to the [Slackdump MCP command help page][mcp-doc].

[mcp-doc]:  https://github.com/rusq/slackdump/blob/master/cmd/slackdump/internal/mcp/assets/mcp.md

## Using as a library

Download:

```shell
go get github.com/rusq/slackdump/v4
```


### Example

```go
package main

import (
  "context"
  "log"

  "github.com/rusq/slackdump/v4"
  "github.com/rusq/slackdump/v4/auth"
)

func main() {
  provider, err := auth.NewValueAuth("xoxc-...", "xoxd-...")
  if err != nil {
      log.Print(err)
      return
  }
  sd, err := slackdump.New(context.Background(), provider)
  if err != nil {
      log.Print(err)
      return
  }
  _ = sd
}
```

See [Package Documentation][godoc].

### Using Custom Logger
Slackdump uses a "log/slog" package, it defaults to "slog.Default()".  Set the
default slog logger to the one you want to use.

If you were using `logger.Silent` before, you would need to
[implement][slog-handler-guide] a discarding [Handler][godoc-slog-handler] for slog.

[slog-handler-guide]: https://github.com/golang/example/blob/master/slog-handler-guide/README.md
[godoc-slog-handler]: https://pkg.go.dev/log/slog#Handler

## FAQ

#### Do I need to create a Slack application?

No, you don't.  Just run the application and EZ-Login 3000 will take
care of the authentication or, alternatively, grab that token and
cookie from the browser Slack session.  See [User's Guide][ug].

#### I'm getting "invalid_auth" error

Run `slackdump workspace new <name or url>` to reauthenticate.

#### How to read the export file?

```shell
slackdump view <ZIP-archive or directory>
```

#### My Slack Workspace is on the Free plan.  Can I get data older than 90-days?

No, unfortunately you can't.  Slack doesn't allow to export data older than 90
days for free workspaces, the API does not return any data before 90 days for
workspaces on the Free plan.

#### What's the difference between "archive", "export" and "dump"?

"Archive" is the new format introduced in v3, it minimises the memory use
while scraping the data and also has a universal structure that can be
converted into export and dump formats at will by using the "convert" command.

"Export" format aims to replicate the files generated when exporting a Slack
workspace for compatibility.

"Dump" format has one channel per file, there's no workspace information nor
any users stored.  Should it be required, one must get users and channels by
running `slackdump list` command.

Behind the scenes slackdump always uses the "archive" file format for all
operations except "emoji" and "list", and converts to other formats on the
fly, removing the temporary archive files afterwards.

## Thank you
Big thanks to all contributors, who submitted a pull request, reported a bug,
suggested a feature, helped to reproduce, or spent time chatting with me on
the Telegram or Slack to help to understand the problem or feature and tested
the proposed solution.

See CONTRIBUTORS.md for the full list of contributors.

Also, I'd like to thank current sponsors:

- [<img class="avatar avatar-user" src="https://avatars.githubusercontent.com/u/9138285?s=60&amp;v=4" width="30" height="30" alt="@malsatin">](https://github.com/malsatin) @malsatin
- [<img class="avatar avatar-user" src="https://avatars.githubusercontent.com/u/836183?s=60&amp;v=4" width="30" height="30" alt="@angellk">](https://github.com/angellk) @angellk

And everyone who made a donation to support the project in the past and keep
supporting the project:

- Davanum S.
- Vivek R.
- Fabian I.
- Ori P.
- Shir B. L.
- Emin G.
- Robert Z.
- Sudhanshu J.

## License

Slackdump is licensed under the [GNU Affero General Public License v3.0 (AGPLv3)](LICENSE).

# Bulletin Board

Messages that were conveyed with the donations:

- 25/01/2022: Stay away from [TheSignChef.com][glassdoor], ya hear, they don't
  pay what they owe to their employees.

[glassdoor]: https://www.glassdoor.com/Reviews/The-Sign-Chef-Reviews-E1026706.htm
