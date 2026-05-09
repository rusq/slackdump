# Replit + Google Drive backup recipe

Run [slackdump](https://github.com/rusq/slackdump) inside a Replit
workspace and stream the resulting `archive/` directory to Google Drive
without ever shipping a 10+ GB tarball through your laptop.

This is useful when:

- Your Slack workspace is large enough that a full archive would not
  fit on the machine you usually run slackdump on.
- You want the archive to live on Drive (cheap cold storage, easy to
  share with teammates) rather than local disk.
- You want the run to be unattended: kick it off in Replit, walk away,
  come back to a directory on Drive that mirrors `archive/` exactly.

The recipe consists of one Node.js script
([`drive-upload-folder.mjs`](drive-upload-folder.mjs)) that walks the
local archive directory, recreates the same folder tree on Drive, and
uploads every file resumably.  A JSON manifest next to the script
tracks per-file progress so you can stop and restart freely.

## Prerequisites

- A Replit workspace with the **Google Drive integration** enabled.
  The script uses Replit's connector proxy
  (`@replit/connectors-sdk`) so you do not need to manage OAuth
  client credentials yourself.
- slackdump has already produced an archive directory in the
  workspace, for example via `slackdump archive -o ./archive`.
- Node.js 18+ (already provided by Replit; the script uses global
  `fetch`).
- A Google Drive folder where the `archive/` subfolder should be
  created.  Its folder ID is the part after `/folders/` in the
  Drive URL.

## Setup

```bash
# From this directory, install the one runtime dependency.
npm install @replit/connectors-sdk
```

## Usage

```bash
DRIVE_PARENT_ID="<your-drive-folder-id>" \
ARCHIVE_DIR="/path/to/archive" \
node drive-upload-folder.mjs
```

The script prints progress every 5 seconds (file count, bytes,
throughput, ETA) and writes a final reconciliation report comparing
local file count and total bytes to what Drive reports.

If the run is interrupted (you stopped it, the workspace got
suspended, the network blipped), just run the same command again.
The manifest tells the script which files are already done and which
were partially uploaded; partial uploads resume from the byte offset
the Drive session reports, not from zero.

## Configuration

All paths and tunables are env vars; defaults are friendly for the
"`archive/` lives next to the script" case:

| Variable | Default | Notes |
| --- | --- | --- |
| `DRIVE_PARENT_ID` | (required) | Google Drive folder ID that will contain the `archive/` subfolder. |
| `ARCHIVE_DIR` | `./archive` | Local archive root to mirror to Drive. |
| `MANIFEST_PATH` | `./archive-upload-manifest.json` | Per-file resume state. Safe to keep across runs. |
| `REPORT_PATH` | `./archive-upload-report.txt` | Final reconciliation report. |
| `PROGRESS_INTERVAL_MS` | `5000` | How often to log progress. |
| `CHUNK_SIZE_MB` | `64` | Resumable upload chunk size. Larger is faster on a clean line, smaller resumes more cheaply on a bad one. |

## How it works

The Replit connector proxy rejects request bodies larger than ~1 MB,
so multipart uploads are not viable for real archive files.  Every
file therefore uses Google Drive's resumable upload protocol:

1. **Initiate session via the proxy.**  The proxy forwards a tiny
   JSON metadata body to Drive and returns a session URL signed by
   Google.
2. **PUT file bytes directly to the session URL.**  This bypasses
   the proxy entirely, so chunk size is bounded only by what your
   network and Drive can sustain.
3. **Save offset to the manifest after every chunk.**  On restart,
   the script asks Drive for the canonical server offset before
   continuing — the manifest is a hint, not the source of truth.
4. **Re-initiate expired sessions transparently.**  Drive expires
   resumable sessions after ~7 days; the script catches `404`/`410`
   and starts a fresh session for the affected file.
5. **Reconcile at the end.**  After the upload loop, the script
   walks the destination on Drive and reports both file count and
   total bytes, so you can verify byte-exact parity before deleting
   the local archive.

## Caveats

- Drive Shared Drives have a 400,000-file limit per drive.  Very
  large slackdump archives can approach this; consider tarring the
  whole `archive/` directory and uploading that single file if you
  hit the cap.
- Files larger than ~5 TB cannot be uploaded to Drive at all (Google
  limit).  This is well above any realistic slackdump archive.
- The script does not delete files from Drive that no longer exist
  locally; it is additive only.
