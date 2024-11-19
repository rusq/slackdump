# Slack Dumper

Purpose:  archive your private and public Slack messages, users, channels,
files and emojis.  Generate Slack Export without admin privileges.

[![Slackdump screenshot](doc/slackdump.webp)](https://github.com/rusq/slackdump/releases/)

**Quick links**:

- [Join the discussion in Telegram](https://t.me/slackdump).
- [Buy me a cup of tea](https://ko-fi.com/rusq_), or use **Github Sponsors**
  button on the top of the page.
- [![Go reference](https://pkg.go.dev/badge/github.com/rusq/slackdump/v3.svg)][godoc]
- How to's:

  - [Mattermost migration][mmost] steps
  - [SlackLogViewerとSlackdumpを一緒に使用する](https://kenkyu-note.hatenablog.com/entry/2022/09/02/090949)
  - [v1 Overview on Medium.com](https://medium.com/@gilyazov/downloading-your-private-slack-conversations-52e50428b3c2)  (outdated)

[godoc]: https://pkg.go.dev/github.com/rusq/slackdump/v3/
[mmost]: doc/usage-export.rst

## Description

Typical use scenarios:

* archive your private conversations from Slack when the administrator
  does not allow you to install applications OR you don't want to use
  potentially privacy-violating third-party tools,
* archive channels from Slack when you're on a free "no archive" subscription,
  so you don't lose valuable knowledge in those channels,
* create a Slack Export archive without admin access, or
* save your favourite emojis.

There are four modes of operation (more on this in [User Guide][ug]):

1. List users/channels
1. Dumping messages and threads
1. Creating a Slack Export in Mattermost or Standard modes.
1. Emoji download mode.

Slackdump accepts two types of input (see [Dumping
Conversations][usage-channels] section):

1. the URL/link of the channel or thread, OR
1. the ID of the channel.

[ug]: doc/README.rst
[usage-channels]: doc/usage-channels.rst

Quick Start
===========

On macOS, you can install Slackdump with Homebrew::

```shell
brew install slackdump
```

On other Operating Systems, please follow these steps:

1. Download the latest release for your operating system from the [releases] page.
1. Unpack the archive to any directory.
1. Run the `./slackdump` or `slackdump.exe` executable (see note below).
1. You know the drill:  use arrow keys to select the menu item, and Enter (or
   Return) to confirm.

[releases]: https://github.com/rusq/slackdump/releases/

By default, Slackdump uses the EZ-Login 3000 automatic login, and interactive
mode.

.. NOTE::
  On Windows and macOS you may be presented with "Unknown developer" window,
  this is fine.  Reason for this is that the executable hasn't been signed by
  the developer certificate.

  To work around this:

  - **on Windows**: click "more information", and press "Run
    Anyway" button.
  - **on macOS**: open the folder in Finder, hold Option and double click the
    executable, choose Run.


## Slackord2: Migrating to Discord

If you're migrating to Discord, the recommended way is to use
[Slackord2](https://github.com/thomasloupe/Slackord2) — a great tool with a
nice GUI, that is compatible with the export files generated by Slackdump.

## User Guide

For more advanced features and instructions, please see the [User Guide][ug].

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
  slack export files.

[SlackLogViewer]: https://github.com/thayakawa-gh/SlackLogViewer/releases
[Slackdump2Html]: https://github.com/kununu/slackdump2html
[slack-export-viewer]: https://github.com/hfaran/slack-export-viewer


## Using as a library

Download:

```shell
go get github.com/rusq/slackdump/v3
```


### Example

```go
package main

import (
  "context"
  "log"

  "github.com/rusq/slackdump/v2"
  "github.com/rusq/slackdump/v2/auth"
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

## FAQ

#### Do I need to create a Slack application?

No, you don't.  Just run the application and EZ-Login 3000 will take
care of the authentication or, alternatively, grab that token and
cookie from the browser Slack session.  See [User's Guide][ug].



#### I'm getting "invalid_auth" error

Go get the new Cookie from the browser and Token as well.

#### Slackdump takes a very long time to cache users

Disable the user cache with `-no-user-cache` flag.

#### How to read the export file?

```shell
slackdump view <ZIP-archive or directory>
```

#### My Slack Workspace is on the Free plan.  Can I get data older than 90-days?

No, unfortunately you can't.  Slack doesn't allow to export data older than 90
days for free workspaces, the API does not return any data before 90 days for
workspaces on the Free plan.

## Thank you
Big thanks to all contributors, who submitted a pull request, reported a bug,
suggested a feature, helped to reproduce, or spent time chatting with me on
the Telegram or Slack to help to understand the issue and tested the proposed
solution.

Also, I'd like to thank all those who made a donation to support the project:

- Vivek R.
- Fabian I.
- Ori P.
- Shir B. L.
- Emin G.
- Robert Z.
- Sudhanshu J.

# Bulletin Board

Messages that were conveyed with the donations:

- 25/01/2022: Stay away from [TheSignChef.com][glassdoor], ya hear, they don't
  pay what they owe to their employees.

[glassdoor]: https://www.glassdoor.com/Reviews/The-Sign-Chef-Reviews-E1026706.htm
