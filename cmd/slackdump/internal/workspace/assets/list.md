# Command: workspace list

The workspace list command displays the Slack workspaces you have previously
authenticated. It supports several output formats:

- **full** (default): Displays workspace names, filenames, and last
  modification dates.
- **bare**: Displays only workspace names, with the current workspace marked by
  an asterisk.
- **all**: Displays all available information, including the team name and user
  name for each workspace.

When the "all" format is selected, Slackdump will query the Slack API to
retrieve the team name and user name for each workspace. This may take some
time, depending on your network speed and the number of workspaces, as it
involves multiple network requests.

