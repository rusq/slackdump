# Troubleshooting

This page covers the most common problems reported by Slackdump users, drawn
from the project's GitHub issue tracker.

## Table of Contents

- [Authentication / Login](#authentication--login)
  - [Browser fails to launch or closes immediately](#browser-fails-to-launch-or-closes-immediately)
  - [Slack rejects the embedded browser ("browser not supported")](#slack-rejects-the-embedded-browser-browser-not-supported)
  - [Google / SSO login blocked in the embedded browser](#google--sso-login-blocked-in-the-embedded-browser)
  - [Login hangs after completing 2FA or SSO (DUO, TOTP, etc.)](#login-hangs-after-completing-2fa-or-sso-duo-totp-etc)
  - [Credentials fail in GitHub Actions or other CI environments](#credentials-fail-in-github-actions-or-other-ci-environments)
  - [`invalid_auth` on enterprise or paid workspaces](#invalid_auth-on-enterprise-or-paid-workspaces)
  - [Token format not recognised](#token-format-not-recognised)
- [Export / Archive](#export--archive)
  - [Export produces no output files](#export-produces-no-output-files)
  - [`slack-export-viewer` crashes or cannot open the ZIP](#slack-export-viewer-crashes-or-cannot-open-the-zip)
  - [Large ZIP files are corrupt ("Bad magic number")](#large-zip-files-are-corrupt-bad-magic-number)
  - [ZIP extraction fails on macOS ("Illegal byte sequence")](#zip-extraction-fails-on-macos-illegal-byte-sequence)
  - [Members array is null/empty in exported JSON](#members-array-is-nullempty-in-exported-json)
  - [Some channels are missing from the archive](#some-channels-are-missing-from-the-archive)
- [File Downloads](#file-downloads)
  - [Filenames with illegal characters fail on Windows](#filenames-with-illegal-characters-fail-on-windows)
  - [Deleted-file attachments cause download errors](#deleted-file-attachments-cause-download-errors)
  - [Avatar download fails with 403 Forbidden](#avatar-download-fails-with-403-forbidden)
- [Rate Limiting](#rate-limiting)
  - [Rate limit errors when listing many channels](#rate-limit-errors-when-listing-many-channels)
  - [`emoji` ignores custom rate-limit flags](#emoji-ignores-custom-rate-limit-flags)
- [Resume / Incremental Backups](#resume--incremental-backups)
  - [Resume freezes after `channel_not_found`](#resume-freezes-after-channel_not_found)
  - [Resume is dramatically slower than a fresh run](#resume-is-dramatically-slower-than-a-fresh-run)
  - [New threads on old messages not picked up by resume](#new-threads-on-old-messages-not-picked-up-by-resume)
- [Performance](#performance)
  - [Archiving thread-heavy channels takes days](#archiving-thread-heavy-channels-takes-days)
  - [Unexpected EOF / i/o timeout after long runs](#unexpected-eof--io-timeout-after-long-runs)
- [Built-in Viewer (`slackdump view`)](#built-in-viewer-slackdump-view)
  - [Viewer returns 404 for downloaded attachments](#viewer-returns-404-for-downloaded-attachments)
  - [Viewer panics / crashes on certain message types](#viewer-panics--crashes-on-certain-message-types)
  - [`gzip: invalid header` on ExFAT drives (macOS)](#gzip-invalid-header-on-exfat-drives-macos)
- [Headless / Docker / CI](#headless--docker--ci)

---

## Authentication / Login

### Browser fails to launch or closes immediately

**Symptoms:** The browser window flashes open and immediately closes, or you see
an error such as `failed to launch browser` or `browser already open`.

**Causes and fixes:**

1. **Another browser session is already open.** Close every instance of Chrome
   (or whichever browser Slackdump uses) and try again. If you're using the
   default `rod`-based headful mode, even a background Chrome process can block
   the automation. See [issue #524].

2. **Missing system libraries on Linux.** On Ubuntu 24.04 and some other
   distributions, the embedded Chromium used by Slackdump may fail to start due
   to missing shared libraries. Try installing them:

   ```bash
   sudo apt-get install -y libnss3 libatk1.0-0 libatk-bridge2.0-0 \
       libcups2 libdrm2 libxkbcommon0 libxcomposite1 libxdamage1 \
       libxfixes3 libxrandr2 libgbm1 libasound2
   ```

   If the problem persists, fall back to the [manual token method](login-manual.md).
   See [issue #293].

3. **Windows: browser closes silently.** On Windows the automated browser
   sometimes exits without showing an error. Try running as Administrator, or
   use the manual token/cookie method instead. See [issue #289].

---

### Slack rejects the embedded browser ("browser not supported")

**Symptoms:** After the browser opens, Slack shows a page saying the browser is
not supported.

**Fix:** Use the manual token/cookie login method. Slack has begun blocking
non-mainstream user-agent strings. Instructions are in
[Manual Authentication](login-manual.md). Particularly common on macOS.
See [issue #328].

---

### Google / SSO login blocked in the embedded browser

**Symptoms:** Google shows "This browser or app may not be secure" and refuses
to complete the OAuth flow.

**Fix:** Google blocks automated browsers. Use the manual token/cookie method
described in [Manual Authentication](login-manual.md), or use the
"Sign In on Mobile" approach to extract the token from the Slack mobile app
(also documented there). See [issue #168].

You could also try using Brave Browser.

---

### Login hangs after completing 2FA or SSO (DUO, TOTP, etc.)

**Symptoms:** You complete MFA in the browser (DUO push, TOTP code, hardware
key, etc.) and the browser redirects to your Slack workspace, but Slackdump
keeps waiting with "Initialising browser, once the browser appears, login as
usual."

**Explanation:** Slackdump watches for a specific token-cookie to appear in the
browser session. Some SSO flows (especially university/enterprise SSO with DUO
Mobile) redirect to a different URL than Slackdump expects, so the detection
never fires.

**Fix:** Use the manual token/cookie method. Log in to your workspace in your
normal browser, extract the `xoxc-` token and `d=` cookie value, and import
them:

```bash
slackdump workspace import <token-file>
```

See [Manual Authentication](login-manual.md) for step-by-step instructions.
See [issue #344].

---

### Credentials fail in GitHub Actions or other CI environments

**Symptoms:** A credential file (`.slackdump` / workspace binary) that works
locally fails in GitHub Actions with `invalid character` errors or garbled
output.

**Cause:** Some CI wrappers (e.g. `faketty`) inject ANSI escape sequences into
stdin/stdout. These sequences corrupt the binary credential file when it is
written or read back.

**Fix:** Use a plain-text token file instead of the binary credential store.
Create a file with just your token:

```
xoxp-...your-token...
```

Then pass it explicitly:

```bash
slackdump workspace import token.txt
```

Store the token as a GitHub Actions Secret and write it to a temp file in
your workflow before calling Slackdump. See [issue #404].

---

### `invalid_auth` on enterprise or paid workspaces

**Symptoms:** Slackdump returns `invalid_auth` even though your token looks
correct, or channel listing fails with `enterprise_is_restricted`.

**Causes:**

- **Wrong token scope.** `xoxs-` session tokens from the browser work broadly.
  `xoxp-` user tokens may lack the API scopes your workspace requires.
- **Enterprise Grid restrictions.** Some Enterprise Grid workspaces restrict
  which API methods third-party tools can call. Only an org admin can lift these
  restrictions. There is no client-side workaround.
- **Token expired or revoked.** Slack session tokens expire when you log out or
  a new device signs in.
- **Slackdump fails to detect enterprise workspace.** You can force enterprise
  API implementation by running with `-enterprise` flag.

**Fix:** Try the "Sign In on Mobile" method from [Manual Authentication](login-manual.md)
to obtain a fresh `xoxc-`/`d=` pair. If `enterprise_is_restricted` persists,
contact your Slack workspace admin. See [issues #40, #273].

---

### Token format not recognised

**Symptoms:** Slackdump rejects a token that you know is valid, e.g.
`token format not supported` or `invalid token`.

**Note:** Slackdump validates token prefixes. Valid prefixes include `xoxp-`,
`xoxs-`, `xoxc-`, and `xoxb-`. OAuth export tokens (`xoxp-`) whose fourth
segment is 32 hex characters (rather than the more common 64) are valid but
were rejected in some older versions. Update to the latest version.
See [issue #562].

---

## Export / Archive

### Export produces no output files

**Symptoms:** Running `slackdump export` with `-time-from` / `-time-to`
completes without errors but produces no ZIP file or an empty one.

**Causes:**

- The date range is too narrow and no messages fall within it. Slack timestamps
  are in UTC — check your local-to-UTC conversion.
- The channel filter (`@file`, channel ID list) resolves to zero channels.

**Fix:** Remove the time range first to confirm data exists, then narrow it.
Double-check that your `@file` contains bare channel IDs (one per line), not
the human-readable table output from `slackdump list channels`. See [issue #428].

---

### `slack-export-viewer` crashes or cannot open the ZIP

**Symptoms:** Third-party tools like `slack-export-viewer` or the official Slack
import tool fail to open a ZIP produced by `slackdump export`.

**Fix:** Use `-type standard` (the default) for `slack-export-viewer`
compatibility. The Mattermost export type (`-type mattermost`) is not
compatible with `slack-export-viewer`. If the viewer still crashes, try
exporting a single channel first to isolate the issue.

Some JSON fields in Slackdump's output differ subtly from official Slack
exports — open an issue with the error message if you encounter a specific
incompatibility. See [issues #63, #222].

---

### Large ZIP files are corrupt ("Bad magic number")

**Symptoms:** A ZIP file around or above 1 GB reports "Bad magic number for
file header" or cannot be opened in Windows Explorer.

**Cause:** This is a ZIP64 boundary issue. Some third-party unzip tools and
older Windows Explorer fail on ZIP64 archives.

**Fix:** Use a modern extraction tool:

```bash
# macOS / Linux
tar xf slackdump_export.zip

# Or use the built-in tool
slackdump tools unzip slackdump_export.zip
```

On Windows, use 7-Zip or WinRAR instead of the built-in Explorer extraction.
See [issue #90].

---

### ZIP extraction fails on macOS ("Illegal byte sequence")

**Symptoms:**

```
error: cannot create __uploads/F08CHQGJL4A/Screenshot 2025-01-22 at 9.10.05ǻAM.txt
       Illegal byte sequence
```

**Fix:** macOS's `unzip` does not handle all UTF-8 filenames. Use `tar` or the
built-in Slackdump tool instead:

```bash
tar xf slackdump_export.zip
# or
slackdump tools unzip slackdump_export.zip
```

See [issue #435].

---

### Members array is null/empty in exported JSON

**Symptoms:** `channels.json` or `groups.json` in the export ZIP contains
`"members": null` even though the channel has members.

**Status:** This was a known issue in older versions. Update to the latest
Slackdump. If the problem persists on the current release, open an issue with
the channel type (public/private/DM). See [issue #220].

---

### Some channels are missing from the archive

**Symptoms:** Running `slackdump archive` without arguments finishes but the
resulting archive is missing channels you can see in Slack.

**Causes:**

- **Visibility:** Slackdump can only see channels the authenticated user is a
  member of, plus public channels it can discover via the API.
- **Rate limiting during channel enumeration.** On very large workspaces,
  channel listing may be cut short by rate limits. Add `-limiter-boost=0` to
  reduce API pressure, or list channels explicitly with an `@file`.

See [issue #544].

---

## File Downloads

### Filenames with illegal characters fail on Windows

**Symptoms:** File downloads fail on Windows with an error about an invalid
filename character (e.g. `?`, `*`, `:`, `<`, `>`).

**Cause:** Slack allows filenames that contain characters that Windows does not
permit in file system paths.

**Fix:** Update to the latest Slackdump — filename sanitisation for Windows was
added to address this. If you are on the latest version and still see the
error, open an issue with the filename. See [issue #521].

---

### Deleted-file attachments cause download errors

**Symptoms:** Export fails with an error about an empty or invalid URL for an
attachment, and the resulting JSON contains malformed attachment data.

**Cause:** When a file is deleted in Slack, it becomes a "tombstone" — the
message still references it, but the download URL is empty.

**Fix:** Add the `-ignore-errors` flag to skip over undownloadable attachments:

```bash
slackdump export -ignore-errors ...
```

See [issue #270].

---

### Avatar download fails with 403 Forbidden

**Symptoms:** User avatar images fail to download with `403 Forbidden`, often
for accounts whose avatar URL contains two consecutive periods.

**Status:** This was a URL-sanitisation bug fixed in a recent release. Update
to the latest Slackdump. See [issue #603].

---

## Rate Limiting

### Rate limit errors when listing many channels

**Symptoms:** You see `slack rate limit exceeded, retry after Xs` errors, or
Slackdump silently produces 0 results for a large workspace.

**Fix:** Reduce API pressure by disabling the rate-limit booster:

```bash
slackdump list channels -limiter-boost=0
```

For archiving, the same flag applies:

```bash
slackdump archive -limiter-boost=0
```

See [issues #1, #12, #28].

---

### `emoji` ignores custom rate-limit flags

**Symptoms:** `slackdump emoji` ignores `-api-config` or `-limiter-boost` and
hammers the API.

**Status:** This was a known issue where the `emoji` subcommand did not
propagate API configuration flags. Check whether your version has a fix; if
not, set a lower concurrency explicitly:

```bash
slackdump emoji -workers=1
```

See [issue #487].

---

## Resume / Incremental Backups

### Resume freezes after `channel_not_found`

**Symptoms:** `slackdump resume` starts, then hangs indefinitely after hitting a
`channel_not_found` error without printing which channel caused it.

**Cause:** A channel that was in the original archive has since been deleted or
archived in Slack.

**Fix:** Stop the process, identify the stale channel (you may need to compare
the channel list in the archive against your current workspace), remove or skip
it, then restart:

```bash
slackdump resume --exclude C012BADCHAN ...
```

See [issue #553].

---

### Resume is dramatically slower than a fresh run

**Symptoms:** `slackdump resume` on the same dataset takes hours or days in
newer versions, but was fast in v3.0.x.

**Status:** A performance regression was introduced in v3.1.x for
resume/incremental mode. Check the [changelog] and update to a version that
includes the fix, or file an issue if it persists on the latest release.
See [issue #560].

[changelog]: https://github.com/rusq/slackdump/blob/master/cmd/slackdump/internal/man/assets/changelog.md

---

### New threads on old messages not picked up by resume

**Symptoms:** After a `slackdump resume -threads` run, brand-new threads that
were started on previously threadless messages are not in the archive.

**Explanation:** Resume with `-threads` only updates threads that already exist
in the archive. It does not re-scan old messages to discover newly created
threads.

**Workaround:** Periodically run a full re-archive of the affected channels, or
use `slackdump archive` without `-threads` on a per-channel basis to catch new
threads. See [issue #584].

---

## Performance

### Archiving thread-heavy channels takes days

**Symptoms:** A channel with many threads (e.g. 20+ threads per day over
several years) is taking an extremely long time to archive.

**Explanation:** Slackdump makes one API round-trip per thread to fetch thread
replies. A channel with 50 000 threads requires 50 000 sequential API calls at
~1 req/s (Slack's Tier 3 rate limit), which is ~14 hours minimum.

**Tips to reduce time:**

- Use `-limiter-burst=5` to allow short bursts within Slack's rate limits.
- Archive only recent messages using `-time-from`:
  ```bash
  slackdump archive -time-from 2024-01-01 C01234ABCDE
  ```
- If you have a previous archive, use `slackdump resume` to only fetch new
  content.

See [issue #543].

---

### Unexpected EOF / i/o timeout after long runs

**Symptoms:** Slackdump runs for hours, then stops with `unexpected EOF` or
`i/o timeout` and does not retry.

**Cause:** Long-lived TCP connections to Slack's API are dropped by
intermediate network infrastructure (NAT, VPN, load balancers). This is
especially common when a VPN is toggled during a run.

**Fix:**

1. Avoid changing network connectivity (toggling VPN, switching Wi-Fi) while a
   long job is running.
2. Use `slackdump resume` to restart from where the job stopped — it is
   designed for exactly this scenario.

See [issues #468, #476].

---

## Built-in Viewer (`slackdump view`)

### Viewer returns 404 for downloaded attachments

**Symptoms:** Files show as downloaded in the archive, but clicking them in the
viewer produces a 404.

**Status:** This was a path-construction bug in the viewer's file-serving
routes. Update to the latest Slackdump. If it persists, open an issue with your
archive format (ZIP / directory / SQLite) and OS. See [issues #554, #561].

---

### Viewer panics / crashes on certain message types

**Symptoms:** The viewer crashes with a nil pointer dereference or template
error when displaying certain channels or messages.

**Known triggers:**

- Messages containing `usergroup` block elements (issue #290).
- Archives that include `.json` attachment files with non-JSON content
  (issue #455).

**Workaround:** Open an issue with the relevant message JSON if the viewer
crashes. As a temporary measure, use `slackdump convert` to convert the archive
to a different format and view it with an external tool.

---

### `gzip: invalid header` on ExFAT drives (macOS)

**Symptoms:** `slackdump view` fails with `gzip: invalid header` when the
archive is stored on an ExFAT-formatted drive (common for USB drives shared
between macOS and Windows).

**Cause:** macOS creates hidden `._` metadata sidecar files on ExFAT volumes.
Slackdump attempts to open these as archive chunks and fails when they turn out
not to be valid gzip data.

**Fix:** Copy the archive to a local APFS or HFS+ volume before running the
viewer. See [issue #473].

---

## Headless / Docker / CI

Running Slackdump in a headless or CI environment requires a pre-exported token
because the browser-based login flow cannot work without a display.

**Recommended approach:**

1. Run the interactive login once on your workstation:
   ```bash
   slackdump workspace new
   ```
2. Export the credentials to a portable file:
   ```bash
   slackdump workspace export myworkspace.toml
   ```
3. Transfer the file to your CI environment and import it at the start of each
   job:
   ```bash
   slackdump workspace import myworkspace.toml
   ```
4. Store the file as an encrypted CI secret (GitHub Actions secret,
   GitLab CI variable, etc.) and write it to disk in your pipeline before
   calling Slackdump.

For Docker, mount the workspace file as a volume:

```bash
docker run --rm \
  -v "$PWD/myworkspace.toml:/workspace.toml" \
  slackdump/slackdump workspace import /workspace.toml && \
  slackdump archive ...
```

> **Note:** Do not use wrapper tools like `faketty` with Slackdump — they
> corrupt binary credential files by injecting ANSI escape sequences.
> See [issue #404].

---

<!-- issue links -->
[issue #1]: https://github.com/rusq/slackdump/issues/1
[issue #12]: https://github.com/rusq/slackdump/issues/12
[issue #28]: https://github.com/rusq/slackdump/issues/28
[issue #40]: https://github.com/rusq/slackdump/issues/40
[issue #63]: https://github.com/rusq/slackdump/issues/63
[issue #90]: https://github.com/rusq/slackdump/issues/90
[issue #107]: https://github.com/rusq/slackdump/issues/107
[issue #168]: https://github.com/rusq/slackdump/issues/168
[issue #220]: https://github.com/rusq/slackdump/issues/220
[issue #222]: https://github.com/rusq/slackdump/issues/222
[issue #270]: https://github.com/rusq/slackdump/issues/270
[issue #273]: https://github.com/rusq/slackdump/issues/273
[issue #289]: https://github.com/rusq/slackdump/issues/289
[issue #290]: https://github.com/rusq/slackdump/issues/290
[issue #293]: https://github.com/rusq/slackdump/issues/293
[issue #328]: https://github.com/rusq/slackdump/issues/328
[issue #344]: https://github.com/rusq/slackdump/issues/344
[issue #404]: https://github.com/rusq/slackdump/issues/404
[issue #428]: https://github.com/rusq/slackdump/issues/428
[issue #435]: https://github.com/rusq/slackdump/issues/435
[issue #455]: https://github.com/rusq/slackdump/issues/455
[issue #468]: https://github.com/rusq/slackdump/issues/468
[issue #473]: https://github.com/rusq/slackdump/issues/473
[issue #476]: https://github.com/rusq/slackdump/issues/476
[issue #487]: https://github.com/rusq/slackdump/issues/487
[issue #521]: https://github.com/rusq/slackdump/issues/521
[issue #524]: https://github.com/rusq/slackdump/issues/524
[issue #543]: https://github.com/rusq/slackdump/issues/543
[issue #544]: https://github.com/rusq/slackdump/issues/544
[issue #553]: https://github.com/rusq/slackdump/issues/553
[issue #554]: https://github.com/rusq/slackdump/issues/554
[issue #560]: https://github.com/rusq/slackdump/issues/560
[issue #561]: https://github.com/rusq/slackdump/issues/561
[issue #562]: https://github.com/rusq/slackdump/issues/562
[issue #584]: https://github.com/rusq/slackdump/issues/584
[issue #603]: https://github.com/rusq/slackdump/issues/603
