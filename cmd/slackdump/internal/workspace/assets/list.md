# Workspace List Command

**List** allows to list Slack Workspaces, that you have previously
authenticated in.  It supports several output formats:
- full (default): outputs workspace names, filenames, and last modification.
- bare: outputs just workspace names, with the current workspace marked with
  an asterisk.
- all: outputs all information, including the team name and the user name for
  each workspace.

If the "all" listing is requested, Slackdump will interrogate the Slack API to
get the team name and the user name for each workspace.  This may take some
time, as it involves multiple network requests, depending on your network
speed and the number of workspaces.
