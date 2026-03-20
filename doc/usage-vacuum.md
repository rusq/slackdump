# Database Vacuum

The `vacuum` tool cleans up the SQLite database by removing duplicate entries
and unreferenced chunks.

## Why use vacuum?

Over time, slackdump databases can accumulate:
- **Duplicate entries**: Same message, user, or file appearing in multiple chunks (from pagination overlap or resume runs)
- **Unreferenced chunks**: Chunks that no longer have any associated records after deduplication

## Usage

```bash
# Preview what would be removed (dry-run)
slackdump tools vacuum /path/to/database.db

# Actually perform the vacuum
slackdump tools vacuum -execute /path/to/database.db

# Target specific operations
slackdump tools vacuum -execute -users /path/to/database.db   # Only remove duplicate users
slackdump tools vacuum -execute -messages /path/to/database.db # Only remove duplicate messages
slackdump tools vacuum -execute -files /path/to/database.db   # Only remove duplicate files
slackdump tools vacuum -execute -chunks /path/to/database.db  # Only remove unreferenced chunks
```

## Flags

| Flag | Description |
|------|-------------|
| `-execute` | Required flag to actually perform the vacuum (without it, only shows counts) |
| `-users` | Only remove duplicate users |
| `-messages` | Only remove duplicate messages |
| `-files` | Only remove duplicate files |
| `-chunks` | Only remove unreferenced chunks |

## How it works

**Deduplication**: Duplicate entries are identified by having the same ID but different CHUNK_ID. When DATA is identical, the older entry (lower CHUNK_ID) is kept and duplicates are removed. This preserves edit history when DATA differs.

**Chunk pruning**: After deduplication, any chunks that no longer have associated records (messages, users, files) are removed.

Each operation runs in its own transaction, so if one fails you can re-run just that specific cleanup without losing progress on others.

## Example

```bash
$ slackdump tools vacuum slackdump.sqlite
Duplicate users: 42
Duplicate messages: 156
Duplicate files: 8
Unreferenced chunks: 23

Total to remove: 229
Run with -execute to perform vacuum.

$ slackdump tools vacuum -execute slackdump.sqlite
Duplicate users: 42
Duplicate messages: 156
Duplicate files: 8
Unreferenced chunks: 23

Removed: 229
```

