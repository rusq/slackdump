# Manual Authentication

[Back to User Guide](README.md)

This page covers how to obtain a Slack token and cookie manually, without the
EZ-Login 3000 browser automation.  This is useful when automated login does not
work (e.g. your workspace enforces strict SSO or MFA policies).

## When to Use Manual Authentication

The recommended method is EZ-Login 3000 (`slackdump workspace new`), which
opens a browser and handles authentication automatically.  Use manual
authentication only as a fallback.

## Method 1: Sign In on Mobile (Recommended Fallback)

This is the easiest manual method and does not require browser developer tools.

1. Open the Slack mobile app on your phone and sign in to the workspace.
2. Tap your workspace name at the top, then go to **Settings**.
3. Scroll down and tap **Sign in on desktop**.
4. Slack will display a link or QR code.  Copy the link (it looks like
   `https://app.slack.com/auth?...`).
5. Open that link in a **desktop browser** — it will sign you into the Slack
   web client.
6. Now follow the [Token](#token) and [Cookie](#cookie) steps below to extract
   the credentials from that browser session.

## Method 2: Browser Developer Tools

### Step 1 — Open Slack in your browser and log in

Open `https://<your-workspace>.slack.com` in your browser and sign in.

### Token

1. Open the browser **Developer Console**:
   - **Firefox**: `Tools → Browser Tools → Web Developer Tools`
   - **Chrome / Edge**: click the three-dot menu → `More Tools → Developer Tools`
2. Switch to the **Console** tab.
3. Paste the snippet below and press **Enter**:

   ```javascript
   JSON.parse(localStorage.localConfig_v2).teams[document.location.pathname.match(/^\/client\/([A-Z0-9]+)/)[1]].token
   ```

4. The token value (starting with `xoxc-`) is printed immediately below.
   Copy and save it.

> **Trouble running the snippet?** See the [alternative method](#alternative-token-extraction) below.

### Cookie

You need the value of the `d` cookie.  There are two ways to get it:

#### Option A — Copy the cookie value directly

1. In the Developer Tools, switch to the **Application** tab (Chrome/Edge) or
   **Storage** tab (Firefox).
2. Expand **Cookies** in the left pane and select your Slack domain.
3. Find the cookie named exactly `d`.
4. Double-click its **Value** field and press `Ctrl+C` / `Cmd+C` to copy.

#### Option B — Export a cookies.txt file

1. Install the [Get cookies.txt LOCALLY](https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc)
   extension (Chrome) or equivalent for your browser.
2. With your Slack tab active, click the extension icon and press **Export**.
3. The file `slack.com_cookies.txt` will be saved to your Downloads folder.
4. Move it to any convenient location.

> **Note:** Option A is sufficient for most workspaces.  Option B (cookies.txt)
> is only necessary if your workspace uses Single Sign-On (SSO) and you keep
> getting `invalid_auth` errors.

## Step 2 — Save the Credentials

Create a file named `.env` (or `secrets.txt` / `.env.txt`) alongside the
`slackdump` executable with these contents:

```
SLACK_TOKEN=xoxc-<...>
SLACK_COOKIE=xoxd-<...>
```

If you used Option B (cookies.txt file), use the file path instead of the
cookie value:

```
SLACK_TOKEN=xoxc-<...>
SLACK_COOKIE=path/to/slack.com_cookies.txt
```

Then import the credentials into Slackdump:

```shell
slackdump workspace import .env
```

It is recommended to delete the `.env` file afterwards.

## Alternative Token Extraction

If the console snippet above does not work, extract the token via the Network tab:

1. Open Developer Tools → **Network** tab.
2. Select **Fetch/XHR** filter.
3. In Slack, open any channel or conversation.  Network requests will appear.
4. Find a request starting with `conversations.history` or `channels.prefs.get`,
   click it, and open the **Headers** (or **Payload**) tab.
5. The `token` field value (starting with `xoxc-`) is your token.

[Back to User Guide](README.md)
