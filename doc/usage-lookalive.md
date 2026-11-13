# Keeping Presence Active (Look Alive)

The `lookalive` tool keeps your Slack presence active (green) while it runs,
like a mouse jiggler, but without touching the mouse or keyboard.

## How it works

The Slack client keeps a websocket connection to the Slack message server.
While you use your computer, it sends small "tickle" activity signals over that
connection; the desktop app also sends them on a timer while you are active in
other applications.  When no activity signals arrive for a while, Slack marks
you as away.

`lookalive` connects to the message server with your saved Slackdump
credentials, the same way the web client does, and sends a tickle every few
minutes (4 minutes ± 25% by default).

An open connection alone is not enough: in testing, with only `lookalive`
connected and no tickles after the first one, Slack marked the user as away
about 30 minutes after the last tickle.  (The Slack API documentation states 10
minutes, so the default interval stays below that.)

## Usage

```bash
# Keep presence active until Ctrl+C
slackdump tools lookalive

# Keep presence active for 8 hours, checking the presence every 10 minutes
slackdump tools lookalive -duration 8h -check-every 10m

# Use a specific saved workspace
slackdump tools lookalive -workspace myworkspace
```

Presence changes for your user are always logged.

## Flags

| Flag | Description |
|------|-------------|
| `-interval` | Base interval between activity signals, minimum 19s (default 4m) |
| `-jitter` | Random jitter fraction applied to the interval, `[0, 1)` (default 0.25) |
| `-duration` | Stop after this duration; `0` runs until interrupted (default 0) |
| `-check` | Check presence with `users.getPresence` after each activity signal |
| `-check-every` | Check presence with `users.getPresence` at this interval, independently of activity signals; `0` disables (default 0) |
| `-set-auto` | Switch presence back to automatic on start, if it was manually set to away |
| `-ping` | Websocket ping interval (default 10s) |
| `-reason` | Activity signal reason: `polling`, `focus`, `mousedown`, `keydown` (default `polling`) |
| `-rtm` | Use `rtm.connect` to get the websocket URL instead of the client message server URL |

## Notes

- If you have manually set yourself as away, Slack keeps showing you as away
  regardless of activity.  Use `-set-auto` to switch back to automatic.
- `lookalive` shows up as one more connected Slack client.  Check your
  workspace policies before using it.
