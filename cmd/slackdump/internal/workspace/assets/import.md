# Workspace Import Command

**Import** allows you to import credentials from a .env or secrets.txt file.

It requires the file to have the following format:
```
SLACK_TOKEN=xoxc-...
SLACK_COOKIE=xoxd-...
```

`SLACK_TOKEN` can be one of the following:

- xoxa-...: app token
- xoxb-...: bot token
- xoxc-...: client token
- xoxe-...: export token
- xoxp-...: legacy user token

`SLACK_COOKIE` is only required, if the `SLACK_TOKEN` is a client type token
(starts with `xoxc-`).

It will test the provided credentials, and if successful, encrypt and save
them to the to the slackdump credential storage.  It is recommended to delete
the .env file afterwards.
