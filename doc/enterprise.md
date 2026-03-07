# Enterprise Workspaces Tips

[Back to User Guide](README.md)

This section applies only in case your Slack Workspace is on Enterprise Plan.

> [!WARNING]
> # Enterprise Workspaces Security Alerts
>
> Depending on your Slack plan and security settings, using Slackdump may
> trigger Slack security alerts and/or notify workspace administrators of
> unusual or automated access/scraping attempts.
> 
> You are responsible for ensuring your use complies with your organisation’s
> policies and Slack’s terms of service.

That said, here are some tips for safe operations without triggering the
scraping alerts:

1. **Always use `-channel-users` flag**.  This will avoid accessing the full
   user list, which may be long and may trigger the logout.  Example:
   ```shell
   slackdump archive -channel-users C01581023
   ```
2. **Avoid full channel listing**.  Collect the URLs or Channel IDs of the
   channels that you need and create a text file with each URL or Channel ID on
   a separate line (see `slackdump help syntax`).  Once you have collected
   everything you need, run:
   ```shell
   slackdump archive -channel-users @list.txt
   ```
   - If you only need to download DMs and the list is short, use
     `-chan-types im`.  If the DM list is large, prefer tip #1 instead.
     For example:
     ```shell
     slackdump archive -chan-types im
     ```

[Back to User Guide](README.md)
