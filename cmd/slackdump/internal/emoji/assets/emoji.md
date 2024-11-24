# Emoji Command
This command allows you to download all the custom emojis from the Slack
workspace.

There are two modes of operation:
- **Standard**: download only the names and URLs of the custom emojis;
- **Full**: Download all the custom emojis from the workspace.

In both modes:
- aliases are skipped, as they just point to the main emoji;
- emoji files and saves in the "emojis" directory within the archive directory
  or ZIP file.


## Standard Mode
In this mode, the command uses the standard Slack API that returns a mapping
of the custom emoji names to their URLs, including the standard Slack emojis.

The output is a JSON file with the following structure:
```json
{
  "emoji_name": "emoji_url",
  // ...
}
```

## Full Mode
In this mode, the command uses Slack Client API to download all information
about the custom emojis.  This includes:
- the emoji name;
- the URL of the emoji image;
- the user display name of the user who created the emoji and their ID;
- the date when the emoji was created;
- it's aliases;
- team ID;
- user's avatar hash.

NOTE: This API endpoint is not documented by Slack, and it's not guaranteed to
be stable.  The command uses the undocumented API endpoint to download the
information about the custom emojis.

It is slower than the standard mode, but slackdump does it's best to do things
in parallel to speed up the process.

The output is a JSON file with the following structure:

```json
{
  "emoji_name": {
    "name": "emoji_name",
    "url": "emoji_url",
    "team": "team_id",
    "user_id": "user_id",
    "created": 1670466722,
    "user_display_name": "user_name",
    "aliases": ["alias1", "alias2"],
    "avatar": "avatar_hash"
  },
  // ...
}
```
