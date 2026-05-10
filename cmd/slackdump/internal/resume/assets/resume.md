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

__NOTE__: Resume is in beta and may not work as expected. Please report any
issues on GitHub.
