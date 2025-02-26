# v3.1.0

- Filenames in Slack Exoprt are dated in the America/Los_Angeles timezone to
  align with the Slack export format;
- 5x faster conversion to Slack export, when using database backend, compared to
  the chunk file backend.
- backend for export, archive and dump formats is changed to database;
- archive and search formats is changed to database;
- universal converter to export for any other format.

# v3.0.0

Gist:
- 2.6x dump speed improvement on channels with threads;
- Support for enteprise workspaces;
- json logging on demand;
- new structured CLI;
- improved TUI for the wizardry with bells and whistles;
- multiple workspaces with ability to switch between them without `-auth-reset`;
- api limits configuration files;
- uninstallation tool;
- slackdump system information tool;
- pgp encryption for traces (under tools);
- search results archival;

## New Archive format

Consider using the new `archive` command to save your workspace data.  You can read about
it in the `slackdump help archive` command and the format it produces in the
`slackdump help chunk` command.

## Viewer

Slackdump V3 introduces a viewer for exported data.  To view the exported data, run:
```
slackdump view <export file or directory>
```

NOTE: search results are not supported by the viewer yet.


## Breaking changes

- `-download` flag renamed to `-files` and is set to "true" by default;
- `-r` flag that allowed to generate text files was replaced by
  `slackdump format` command.

## New features

- Completely rewritten CLI, based on `go` command source code (see
  [Licenses][1]);
- Default API limits can be overridden with configuration file (use
  `-config <file>`);
- Slack Workspaces:
    - Slackdump remembers credentials for multiple Slack Workspaces;
    - It is possible to select the "_Current_" Workspace using the
      `workspace select` command;
    - The "_Current_" workspace can be overridden by providing the `-w <name>`
      flag.
- Slackdump `archive` mode allows to dump the entire workspace into a directory
  of chunk files.
- Slackdump `convert` mode allows to convert chunk files into other formats,
  such as Slack export format, or Slackdump format.

## Changes

- Default output location (**BASE_LOC** environment variable), if not set by the
  user, defaults to the ZIP file "slackdump_YYYYMMDD_HHmmSS.zip", where
  `YYYYMMDD` is the current date (for example `20221103`) and `HHmmSS` is the
  current time with seconds (for example `185803`);
- To reset all authentication data (similar to old `-auth-reset`), run
  `slackdump workspace delete -a -y` where `-a` is for "all" and `-y` to
  answer "yes" to all questions;
- Flag `-user-cache-file` was removed.
- Slackdump does not cache users on each startup, to speed up execution, it
  users lazy caching. For example, when the list command is requested.  The
  user cache behaviour can still be controlled by `-no-user-cache` and
  `-user-cache-retention` flags.


# Library changes

## Deprecation of Dump* functions

## Slackdump Core

- `Options` reorganised, API limits are extracted into a Limits variable. Tier
  limits are extracted to TierLimits, and are accessible via `Limits.Tier2` and
  `Limits.Tier3` variables. Requests limits are accessible via
  `Limits.Request`.
- `Session.SetFS` method is removed, set the filesystem in `Options.Filesystem`.
- Introduced `Close()` interface method on `fsadapter.FS`.  `fsadapter.Close` is
  removed.

## Licenses

[1]: #licenses
