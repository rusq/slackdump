# Security Policy

## Reporting a vulnerability

Please do not include vulnerability details, archives, logs, access tokens, or
other sensitive information in an unencrypted GitHub issue.

Encrypt your report with the developer's PGP key, then attach the encrypted
file to a GitHub issue:

```console
slackdump tools encrypt -a report.md report.md.asc
```

This command encrypts the report for the developer; only the developer can
decrypt it. The developer key fingerprint is:

```text
3375 30DD 887F B454 C4FC  6E7F 23B9 DBD7 FAE5 4FCD
```

Include enough information to reproduce the issue safely: affected version,
platform, impact, and a minimal proof of concept where possible. We aim to
acknowledge reports within seven days and will coordinate a fix and disclosure
timeline with the reporter.

GitHub private vulnerability reporting is not currently enabled for this
repository, so encrypted issue attachments are the reporting channel.
