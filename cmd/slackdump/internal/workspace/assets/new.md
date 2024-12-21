# Command "workspace new"

This command authenticates to a new Slack workspace and saves the
authentication credentials.

- If a workspace with the same name already exists, Slackdump will prompt you
  to either overwrite the existing workspace or cancel the operation.
- If no workspace name is provided, the command will create or replace the
  "default" workspace.

**Note**:  If you're migrating from Slackdump v2, you will only have a "default"
workspace by default. You can either continue using it as your default
workspace or create a new one. Later, you can switch between workspaces using
the `workspace select` command.

## Usage

```shell
slackdump workspace new <workspace name or url>
```

For example:
```shell
slackdump workspace new https://ora600.slack.com
```
