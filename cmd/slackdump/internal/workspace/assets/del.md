# Command: "workspace del"
The `workspace del` command removes the Slack workspace login information
(effectively "forgetting" the workspace).

Once deleted, you will need to re-authenticate by running the `workspace new`
command to log in to that workspace again.

Slackdump will prompt you for confirmation before deleting the workspace. To
bypass this confirmation, use the `-y` flag.

