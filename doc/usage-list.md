# Listing Users and Channels

[Back to User Guide](README.md)

The `list` command retrieves and displays users or channels visible to your
account.

## List Users

```shell
slackdump list users
```

Prints all workspace users.  Results are also saved as a JSON file in the
output directory.

## List Channels

```shell
slackdump list channels
```

Prints all conversations visible to your account, including public channels,
private channels, group messages, and direct messages.

### Resolve Usernames

Add `-resolve` to replace user IDs with display names in the channel list
(useful for DM and group message entries):

```shell
slackdump list channels -resolve
```

> **Note:** resolving usernames fetches all workspace users, which can be slow
> on large workspaces.

### Filter by Channel Type

Use `-chan-types` to restrict which conversation types are returned:

```shell
# Only public channels
slackdump list channels -chan-types public_channel

# DMs and group messages only
slackdump list channels -chan-types im,mpim
```

Available types: `public_channel`, `private_channel`, `im`, `mpim`.
Default: all four types.

### Only Channels You're a Member Of

```shell
slackdump list channels -member-only
```

## Output Format and Saving

By default the output is printed to the screen **and** saved as a JSON file in
the output location.  Use these flags to control the behaviour:

| Flag | Description |
|------|-------------|
| `-o location` | Directory or ZIP file to save results (default: `slackdump_<ts>.zip`) |
| `-no-save` | Print to screen only, do not save a file |
| `-q` | Quiet mode — suppress screen output (file is still saved) |

## Sample Output

```
ID           Arch  What
CHXXXXXXX    -     #everything
CHXXXXXXX    -     #everyone
CHXXXXXXX    -     #random
DHMAXXXXX    -     @slackbot
DNF3XXXXX    -     @alice
DLY4XXXXX    -     @bob
```

> **Large workspaces:** listing all channels in a 20,000-channel workspace can
> take up to an hour because Slack enforces strict API rate limits.

## Key Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-chan-types value` | `mpim,im,public_channel,private_channel` | Filter channel types |
| `-member-only` | — | Only return channels the current user belongs to |
| `-resolve` | — | Resolve user IDs to display names |
| `-no-save` | — | Do not save results to a file |
| `-q` | — | Suppress screen output |
| `-o location` | `slackdump_<ts>.zip` | Output location |
| `-workspace name` | current | Override the active workspace |

[Back to User Guide](README.md)
