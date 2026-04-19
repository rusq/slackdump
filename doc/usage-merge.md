# Merging Archives

The `merge` tool combines one or more Slackdump sources from the **same
workspace** into an existing database archive.

## Why use merge?

Common scenarios:

- You ran separate `archive` or `export` jobs for different channels and want
  to consolidate them into one database.
- You have archives from different time windows and want a single unified file.
- You are combining archives produced by different users of the same workspace.

> [!WARNING]
> Always make a backup of the target database before running merge.
> The operation modifies the target in place and cannot be undone.

## Usage

```bash
slackdump tools merge <target database> <source1> [source2 ...]
```

The target database must already exist (create one with `slackdump archive`
first).  Sources can be database archives, chunk-file directories, or standard
Slack export ZIPs — all from the **same** workspace.

```bash
# Check compatibility without modifying anything
slackdump tools merge -check ./combined.db ./archive-jan ./archive-feb

# Merge, skipping file attachments
slackdump tools merge -files=false ./combined.db ./archive-jan ./archive-feb

# Merge everything including files and avatars (default)
slackdump tools merge ./combined.db ./archive-jan ./archive-feb
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-check` | `false` | Verify that sources are compatible with the target; do not merge |
| `-files` | `true` | Copy file attachments from sources into the target |
| `-avatars` | `true` | Copy user avatars from sources into the target |

## Deduplication after merge

`merge` does **not** deduplicate.  If the sources overlap in time (e.g. a
resume look-back window), the target may contain duplicate rows.  Run
`slackdump tools dedupe` on the target afterwards:

```bash
slackdump tools dedupe -execute ./combined.db
```

See [Database Dedupe](usage-dedupe.md) for details.
