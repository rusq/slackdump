# Automatic (Browser-Based) Login

[Back to User Guide](README.md)

Slackdump's automatic login is handled by the
[`slackauth`](https://github.com/rusq/slackauth) library, which drives a
browser via the [Rod](https://go-rod.github.io/) / CDP protocol to capture the
Slack session token and cookie on your behalf.

Run the following command to add a new workspace:

```bash
slackdump workspace new
```

You will be asked for the workspace name (the part before `.slack.com`), then
prompted to choose one of four login methods described below.

## Login Methods

### Interactive (recommended for most users)

A clean browser window opens on the Slack login page.  Log in as usual —
including any SSO, MFA, or company identity provider steps.  The browser
closes automatically once the session token and cookie have been captured.

Choose **Interactive** unless your workspace uses Google Authentication (use
**User Browser** instead) or you know your email/password login works
headlessly.

### User Browser

Instead of launching a bundled browser, Slackdump opens **your own installed
browser** (Chrome, Firefox, etc.) on the Slack login page.  Your existing
browser profile is used, which means Google Authentication and other flows that
block embedded browsers will work.

When prompted, select the browser you want to use from the list of detected
browsers.

### Automatic (Headless)

Slackdump automates the email + password login flow entirely without opening a
visible browser window.  You will be prompted to enter your email and password
in the terminal.  If Slack sends a confirmation code to your email, you will
also be asked to enter that.

**Limitations:** only works with plain email/password workspaces.  Does not
support SSO, Google, or passwordless (OTP-only) workspaces.

### QR Code (Sign in on Mobile)

Use this method when other browser-based methods are blocked (e.g. Google Auth
blocks the embedded browser, or your workspace enforces strict SSO policies).
It uses the Slack "Sign in on mobile" QR code flow, which still relies on Rod
internally to exchange the QR image for a session token.

**Steps:**

1. In the logged-in Slack desktop app or web client, click your **workspace
   name** in the upper-left corner.
2. Choose **Sign in on mobile**.
3. Slack displays a QR code.  **Right-click the QR code image** and choose
   **Copy Image**.
4. Switch to the Slackdump terminal — it will be showing a text field titled
   "Paste QR code image data into this field".
5. Paste the copied image data into that field and press **Enter**.

Slackdump exchanges the base64-encoded image with Slack's API and captures the
resulting session token and cookie automatically.

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| Browser fails to launch or closes immediately | Close any existing Chrome/browser processes, then retry. See [Troubleshooting](troubleshooting.md#browser-fails-to-launch-or-closes-immediately). |
| Slack shows "browser not supported" | Switch to **User Browser** or use [manual login](login-manual.md). |
| Google / SSO login blocked | Use **User Browser** or **QR Code** method. |
| Login hangs after completing 2FA / SSO | Use **QR Code** method or fall back to [manual login](login-manual.md). |
| Running in Docker / CI | Browser-based login requires an interactive display. Use [manual login](login-manual.md) instead. |

[Back to User Guide](README.md)
