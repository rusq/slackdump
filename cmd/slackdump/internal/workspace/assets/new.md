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
### Free and Standard Workspaces

```shell
slackdump workspace new <workspace name or url>
```

For example:
```shell
slackdump workspace new https://ora600.slack.com
```

### Enterprise Grid Workspaces

The `<name>.enterprise.slack.com` is the name of your — let's call it — an
_enterprise instance_.  An enterprise instance may have one or more workspaces.
The workspace will have the URL like `<workspace_name>.slack.com`.

For example:
```
- gm-hq.enterprise.slack.com
  - opel.slack.com
  - chevrolet.slack.com
  - holden.slack.com
```

You need to specify the workspace URL or "workspace_name", not the enterprise instance name.

To find out <workspace_name>, you can do the following:
1. Log in to the enteprise workspace using Slack Client or Slack Web UI;
2. Click on the top menu (the one in the upper left corner, looks like
   "**YourCompany  v**";
3. Choose Tools->Customise workspace;
4. It will prompt you for the workspace - pick one and click "Open";
5. The browser window or tab will open containing the setting for the
   workspace, i.e. Emoji customisation.
6. The URL in the browser address bar is the URL of the workspace, in the form
   `<workspace_name>.slack.com`;
7. Grab and use "workspace_name" or "https://workspace_name.slack.com" to
   authenticate.

