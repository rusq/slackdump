# Command: "workspace import"

The `workspace import` command allows you to import credentials from a `.env`
or `secrets.txt` file.

The file should have the following format:
```shell
SLACK_TOKEN=xoxc-...
SLACK_COOKIE=xoxd-...
```
The command will test the provided credentials, and if successful, it will
encrypt and save them to Slackdump's credential storage. It is recommended to
delete the .env or secrets.txt file after the import to ensure security.

Slackdump will detect the name of the workspace automatically and Select it as
current.

**SLACK_TOKEN** can be one of the following types:

- xoxa-...: App token
- xoxb-...: Bot token
- xoxc-...: Client token
- xoxe-...: Export token
- xoxp-...: Legacy user token

**SLACK_COOKIE** is required only if the `SLACK_TOKEN` is a client token
(`xoxc-...`).


