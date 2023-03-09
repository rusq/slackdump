# Contributions

This directory contains scripts and helper tools that has been contributed to
this project.  See the [catalogue](#catalogue) for a list of the available
tools.

For more information on how to contribute, see the
[contributing](#contributing).


## Catalogue
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

### Incremental Workspace Backups

- Author: [@levigroker](https://github.com/levigroker)
- Path: [incremental_backup/dump.sh](incremental_backup/dump.sh)
- Source: [link](https://gist.github.com/levigroker/fa7b231373e68269843aeeee5cc845a3)
- Description: Dumps messages and attachments for selected 1-1 direct messages, and selected named
channels and group PMs, from the authenticated Slack workspace. Subsequent runs will
fetch only the new content since the previous run.

## Contributing

If you have a script or tool that you think would be useful to others, please
consider contributing it to this project.  To do so, please follow these steps:

1. Fork the project;
2. Add your script or tool to the `contrib` directory;
3. Add the tool description to the `catalogue.yaml` file;
4. Submit a pull request.
