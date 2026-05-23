# Resume command

Resume allows to continue the archive where you left off.

It may be useful in the following situations:
- Slackdump crashed or was interrupted;
- You want to add data to an existing archive.

Please note that archive must be in database format (default for "archive"
command).

### Resuming Export, Chunk, or Dump formats.
If you want to resume a Slack Export, Chunk, or dump formats, follow these
steps:

1. Convert it to database format first:
    
   ```plaintext
   slackdump convert -format database <your-export .zip or directory>
   ```

   This will create a new directory with Slackdump database, i.e.
   `slackdump_20241231_150405`.
2. Resume the archive:

   ```plaintext
   slackdump resume slackdump_20241231_150405
   ```

   This will continue the archive where you left off.
3. When Slackdump finishes, the archive will be updated with the
   latest data.  Convert it back to the desired format:
   ```bash
   slackdump convert -format <your-format> slackdump_20241231_150405
   ```

### Deduplicating resume overlap.

Resume uses a lookback window to avoid missing edits, replies, and other
late-arriving changes. This can re-fetch unchanged entities that are already in
the archive. Use `-dedupe` to remove identical duplicate messages, users,
channels, channel users, files, and now-empty chunks after a successful resume:

```plaintext
slackdump resume -dedupe slackdump_20241231_150405
```

Deduplication runs only after resume finishes successfully. To preview or run
the same cleanup manually later, use `slackdump tools dedupe`.

### Skipping stale entities.

Long-lived archives accumulate channels and threads that have not received any
new activity for weeks or months. Resuming such archives spends most of its
wall-clock time fetching the first page of `conversations.replies` for
thread parents that will never get another reply, which also burns
rate-limit budget on `conversations.replies` and is the most common cause of
Slack 429 retries during resume.

`-skip-stale-threads` and `-skip-stale-channels` filter dormant entities out
of the entity list **before** any API call fires. Both flags accept an ISO
8601 duration (e.g. `p21d`, `p7d`, `p2w`) and default to disabled when not
set:

```plaintext
# Skip threads whose latest reply is older than 21 days; channels untouched.
slackdump resume -threads -skip-stale-threads p21d <archive>

# Skip dormant channels in addition. Pair with a periodic full-sweep run
# (e.g. a daily cron without the skip-stale flags) so dormant channels
# are still revisited and resurrections are surfaced.
slackdump resume -threads -skip-stale-threads p21d -skip-stale-channels p21d <archive>
```

The two flags are independent. Skipping stale **channels** is the more
aggressive trade because brand-new top-level messages in a skipped channel
will not be picked up until that channel is included again — pair with a
periodic full sweep. Skipping stale **threads** is conservative: dormant
thread parents are very unlikely to receive new replies, and a full sweep
will still catch them.
If stale filters skip every resume candidate, resume exits successfully as a
no-op before setting up a Slack API session.

__NOTE__: Resume is in beta and may not work as expected. Please report any
issues on GitHub.
