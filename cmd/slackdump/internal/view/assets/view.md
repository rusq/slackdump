# View Command

The `view` command allows you to view the contents of an archive, 
export, or dump directory or ZIP file.

It is a read-only command that does not modify the contents of the
specified directory or file.

Viewer supports displaying downloaded images, videos as well as remote
content.

## Usage

```bash
slackdump view <directory_or_file>
```

If you experience problems viewing, run the viewer with DEBUG mode
enabled, and report the violating message to the GitHub Issues page.

```bash
DEBUG=1 slackdump view <directory_or_file>
```

It is recommended that you remove all sensitive information from the
JSON before sharing it, and also, to encrypt your message, you can use
the `slackdump tools encrypt` command, for example:

```bash
cat your_message.txt | slackdump tools encrypt > encrypted_message.txt
```

This will encrypt it using the embedded GPG public key, and can only be
encrypted by the author.
