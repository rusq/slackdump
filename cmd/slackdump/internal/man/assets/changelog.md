# v2.3.0

Breaking changes:
- legacy command line interface moved under "v1" command, so if you need to
  use the old CLI, instead of `./slackdump`, run `./slackdump v1`.  The
  legacy CLI will be phased out:  deprecated in v2.4.0, and removed in v2.5.0.
- download flag is set to "true" by default.

New features:
- Completely rewritten CLI, based on `go` command source code (see
  [Licenses](#licenses)).
- Default API limits can be overridden with configuration file (use
  `-config <file>`).
- Slack Workspaces:
  - Slackdump remembers credentials for multiple Slack Workspaces;
  - It is possible to set the "current" Workspace;
  - "Current" workspace can be overridden by providing the "-w \<name\>" flag. 

Changes
- Default output location (BASE_LOC environment variable), if not set by the
  user, defaults to the ZIP file "slackdump_YYYYMMDD_HHmmSS.zip", where
  YYYYMMDD is the current date (for example 20221103) and HHmmSS is the current
  time with seconds (for example 185803). 

## Licenses
