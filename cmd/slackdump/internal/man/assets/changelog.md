# v2.3.0

## Breaking changes

- legacy command line interface moved under "v1" command, so if you need to
  use the old CLI, instead of `./slackdump`, run `./slackdump v1`. The legacy
  CLI will be phased out:  deprecated in v2.4.0, and removed in v2.5.0;
- `-download` flag renamed to `-files` and is set to "true" by default;
- `-r` flag that allowed to generate text files was replaced by
  `slackdump convert` command.

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
