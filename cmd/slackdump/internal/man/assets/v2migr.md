# Migration from V2 to V3

## Authentication

V2 supported a single Slack workspace.  If your Slackdump is already logged in,
this workspace will be listed as "default" in the `slackdump workspace list`
output.  You can continue to use this workspace in V3.

If you were using `.env` or `secrets.txt` file for authentication, you need to
run: `slackdump workspace import <filename>` to import the workspace.

## Commands

V3 introduces a new command structure.  The commands are now grouped into
categories.  Usually the mapping is straightforward, i.e. `slackdump
-list-users` becomes `slackdump list users`. The `help` command will show you
the new command structure.

Notable change is that files are downloaded by default. If you want to disable
this behaviour, you need to specify `-files=false` flag, i.e.
```
slackdump export -files=false
```

## Entity list

If you were using the individual timestamps for channels, the syntax has changed to use comma
delimiters "," instead of pipe "|".  For example, to limit the export for channel C123 to
January 2024, you should use:
```
slackdump export C123,2024-01-01T00:00:00,2024-02-01T00:00:00
```
(instead of `C123|2024-01-01T00:00:00|2024-02-01T00:00:00`)

## Suggestions

Suggestions to add to this document are welcomed, please open an issue or a pull request.
