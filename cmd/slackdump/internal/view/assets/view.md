# View Command

The `view` command allows you to view the contents of an archive, 
export, or dump directory or ZIP file.

It is a read-only command that does not modify the contents of the
specified directory or file.

Viewer supports displaying downloaded images, videos as well as remote
content.

By default the viewer accepts only localhost, loopback addresses, and the
specific host named by `-listen`. To expose it through a LAN address or a
reverse proxy, explicitly repeat `-allow-host <hostname-or-ip>` for each Host
header that should be accepted. Do not use an unrestricted wildcard host.

The viewer uses a side panel for threads and user profiles, keeps the active
channel highlighted while navigating, and reports connection problems if the
local viewer server becomes unreachable.

Use **Settings** beside the sidebar heading to choose whether conversations open
at the **Oldest** or **Latest** message. Oldest is the default. Changes apply the
next time you open a conversation; message order remains chronological. The
**Jump to latest** button moves to the bottom without changing your preference.
Message links and browser history navigation take precedence over this setting.

Preferences are stored in this browser's local storage and persist across viewer
restarts. Archives opened at the same viewer address share the preference;
different browsers, hostnames, ports, or protocols have separate preferences.
Clearing browser site data resets the preference, and private browsing or blocked
storage may prevent persistence. Settings do not modify the archive and are not
included in converted static HTML.

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
