# Contributions

This directory contains scripts and helper tools that has been contributed to
this project.  See the [catalogue](#catalogue) for a list of the available
tools.

For more information on how to contribute, see the
[contributing](#contributing).


## Catalogue
### Slackdump Export URL rewriter (for Discord Migration)

- Author: [@dannyadair](https://github.com/dannyadair)
- Path: [rewrite_slackdump__urls](rewrite_slackdump__urls)
- Source: [link](https://github.com/rusq/slackdump/issues/399#issuecomment-3393201727)
- Description: A script to rewrite file URLs in the Slackdump export.

> Just drop it in the export folder and run it.  It will rewrite the JSON files
> to use a new host/port - I'm just setting it to localhost and run a simple web
> server on the computer running Slackord2 to serve them up.


### Incremental Workspace Backups

- Author: [@levigroker](https://github.com/levigroker)
- Path: [incremental_backup/dump.sh](incremental_backup/dump.sh)
- Source: [link](https://gist.github.com/levigroker/fa7b231373e68269843aeeee5cc845a3)
- Description: Dumps messages and attachments for selected 1-1 direct messages, and selected named
channels and group PMs, from the authenticated Slack workspace. Subsequent runs will
fetch only the new content since the previous run.


### Example: Find matches and print dates and text

- Author: [@fitzyjoe](https://github.com/fitzyjoe)
- Path: [messages_json_parsing/jq/find_matches_print_dates_and_text.sh](messages_json_parsing/jq/find_matches_print_dates_and_text.sh)
- Description: Finds messages that match a regex and prints the date and text.

### Example: Print messages

- Author: [@rusq](https://github.com/rusq)
- Path: [messages_json_parsing/jq/print_messages.sh](messages_json_parsing/jq/print_messages.sh)
- Description: Prints user ID and a message

### Example: Print messages

- Author: [@rusq](https://github.com/rusq)
- Path: [messages_json_parsing/python/print_messages.py](messages_json_parsing/python/print_messages.py)
- Description: Prints user ID and a message
## Contributing

If you have a script or tool that you think would be useful to others, please
consider contributing it to this project.  To do so, please follow these steps:

1. Fork the project;
2. Add your script or tool to the `contrib` directory;
3. Add the tool description to the `catalogue.yaml` file;
4. Submit a pull request.
