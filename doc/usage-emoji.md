# Downloading Emojis

[Back to User Guide](README.md)

The `emoji` command downloads all custom emojis from the workspace.

## Modes

There are two modes of operation:

| Mode | Flag | Description |
|------|------|-------------|
| **Standard** (default) | — | Downloads emoji names and image URLs using the standard Slack API |
| **Full** | `-full` | Uses the Slack Edge API to fetch rich metadata (creator, date, aliases) — ~2.3× slower |

> **Note:** Full mode uses an undocumented Slack API endpoint and may be less
> stable than standard mode.

## Quick Start

```shell
# Download to a timestamped ZIP (default)
slackdump emoji

# Download to a specific ZIP file
slackdump emoji -o my_emojis.zip

# Download to a directory
slackdump emoji -o emoji_dir

# Full mode with extended metadata
slackdump emoji -full -o my_emojis.zip
```

On Windows replace `./slackdump` with `slackdump`.

## Output Structure

```
.
├── emojis/
│   ├── foo.png
│   ├── bar.png
│   └── baz.png
└── index.json
```

- **`index.json`** — index of all emojis returned by the API.  In standard
  mode this is a `{ "name": "url" }` map; in full mode each entry includes
  creator info, creation date, and aliases.
- **`emojis/`** — downloaded emoji image files, named `<emoji_name>.png`.

Aliases are **skipped** — only the original emoji files are downloaded.  To
find the original for an aliased emoji, search `index.json` for the alias name;
the `url` field will contain `alias:<original_name>`.

## Key Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-o location` | `slackdump_<ts>.zip` | Output directory or ZIP file |
| `-full` | — | Use Edge API for rich metadata |
| `-ignore-errors` | `true` | Skip failed downloads instead of stopping |
| `-workspace name` | current | Override the active workspace |

[Back to User Guide](README.md)
