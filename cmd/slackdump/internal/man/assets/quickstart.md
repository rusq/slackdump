# Quick Start

There are two ways to use Slackdump:
1. Wizard Mode;
2. Command Line Mode.

## Wizard Mode
1. Start Slackdump without parameters, and choose __Run wizard__.  Alternatively,
   you can start wizard by running `slackdump wiz`).
2. Add a new workspace:
   1. In the main menu, choose __Workspace => New__
   2. Select __Login In Browser__, and type your workspace name or paste a URL.
   3. If your workspace uses Google Authentication, select __User Browser__ and
      pick the one installed on your system.
   4. In all other cases, choose __Interactive__ mode.
   5. The browser will open, login as usual. Choose "Open this workspace in
      browser." The new workspace is automatically selected, and ready to use.
      (You can switch between workspaces using the __Workspace__ menu).  If
      Slack Web Client opens in another tab, do the following:
      - copy the URL,
      - switch to the original tab
      - paste it and press Enter.

      Slackdump will continue the login process.
   6. Exit the Workspace menu by choosing __Exit__ to exit to workspace menu
      and then __<< Back__ to return to the main menu.
3. Select __Archive__ or __Export__ items to save your workspace data. You can
   configure the following options:
     - specify the list of channel IDs to include or exclude in the "Archive
       Options" (or "Export Options").  The list supports URLs, that you can
       copy and paste from the Slack client.
     - define the Time range in the "Global Configuration".

     The difference between Options and Global Configuration is that the Options
     affect only the current command, while the Global Configuration affects all
     commands.
4. The data is saved by default in the directory or ZIP file that starts with
   `slackdump-<timestamp>`.
5. To view your data, you need to use the Command Line Mode, see item 3 below.

## Command Line Mode
1. Add a workspace `slackdump workspace new <workspace_name>`.  The browser
   will open, login as usual.
2. Run `slackdump archive` or `slackdump export` to save your workspace data.
3. Run `slackdump view <archive name>` to view the data.

## Fallback to Manual login
