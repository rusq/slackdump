# internal/viewer — Agent Guide

> For project-wide conventions (error sentinels, mock generation, build commands, interface naming,
> logging), see `.github/copilot-instructions.md`. This file covers only invariants local to the
> `viewer` package.

## Package map

| File | Responsibility |
|---|---|
| `viewer.go` | `Viewer` struct, `New()`, HTTP server setup, route registration, channel classification |
| `handlers.go` | All HTTP handlers, `mainView` data struct, `setConversation` helper |
| `filestorage.go` | `fileByIDStorage` optional extension interface + `fileByID` helper |
| `template.go` | Template compilation (`initTemplates`), FuncMap, sender classification |
| `templates/index.html` | All HTML template defines (full page + HTMX partials + JS) |
| `templates/styles.html` | All CSS (single `hx_css` define, CSS variables, dark mode) |

---

## Invariants

### 1. HTMX dual-mode rendering

Every handler that can be reached by both a direct browser URL and an HTMX partial swap **must**
branch on `isHXRequest(r)`:

- **HTMX request** → render the relevant partial template (e.g. `hx_conversation`, `hx_canvas`).
- **Direct / deep-link request** → render full `index.html` so the sidebar and layout are present.

Failing to do this breaks deep-link bookmarkability.

### 2. `setConversation` must be called in every channel-rendering handler

`setConversation(page, ci)` is the **only** place that sets both `page.Conversation` and
`page.CanvasAvailable`. It must be called in every handler that renders a view containing a channel
header, including deep-link fallback paths.

Current call sites: `channelHandler`, `threadHandler`, `canvasHandler`.  
If you add a new handler that renders a channel, add a `setConversation` call.

### 3. Optional extension interface — never extend `source.Storage` directly

When internal viewer code needs a capability that not all storage backends provide, use the pattern
in `filestorage.go`:

1. Declare an **unexported** interface in the viewer package (`fileByIDStorage`).
2. Use a runtime type assertion to check if the concrete storage implements it.
3. Degrade gracefully (return `fs.ErrNotExist`-wrapped error) if it does not.

Do **not** add the method to the public `source.Storage` interface — that is a breaking change for
third-party implementations.

### 4. Renderer must be assigned before `initTemplates`

`v.r` (the `renderer.Renderer`) must be fully initialised before `initTemplates(v)` is called in
`New()`, because the `rendertext` and `render` template functions close over `v.r`.  
Reordering these two steps causes a nil-dereference at first template execution.

### 5. Path component safety

All path components extracted from the URL (file IDs, timestamps, filenames) must be validated with
`isInvalid(pcomp)` before use in file system operations. `isInvalid` rejects `..`, `~`, `/`, `\`.

### 6. Canvas — graceful degradation, never error

Canvas support degrades silently when the storage backend does not implement `fileByIDStorage`.
The tab is shown **disabled** (HTML `disabled` attribute, CSS `.disabled`), never hidden or
erroring. `canvasContentHandler` logs at `DEBUG` level before returning 404 when the file is
absent or the storage does not support `FileByID`.

### 7. Canvas tab visibility

The `tab_list` partial (and therefore the tab bar) is only rendered when
`canvas_present .Conversation` returns true, i.e. `Channel.Properties.Canvas.FileId != ""`.
`Properties` is a pointer — it can be nil; `canvas_present` must guard against that (current
implementation does via the template func in `template.go`).

### 8. Template iframe sandbox

The canvas iframe uses `sandbox="allow-same-origin"` only — no scripts from canvas content may
execute. Do not add `allow-scripts` to this attribute.

### 9. Tab CSS — `box-shadow` not `border-bottom`

The active-tab indicator uses `box-shadow: inset 0 -3px 0 var(--primary-color)`.  
The inactive tab pre-reserves space with `box-shadow: inset 0 -3px 0 transparent`.  
This avoids layout shift on selection. Do not replace with `border-bottom`.

### 10. CSS — always use variables, never hardcoded colours

All colours must reference the CSS custom properties defined in `:root` (e.g. `--primary-color`,
`--bg-color`). Dark mode is driven purely by `@media (prefers-color-scheme: dark)` overriding
those variables. Hardcoded colour values break dark mode.

---

## Adding a new handler — checklist

1. Implement the handler as a method on `*Viewer` in `handlers.go`.
2. Check `isHXRequest(r)` and render the appropriate partial or full `index.html`.
3. Call `setConversation(page, ci)` if the handler renders any channel view.
4. Validate all URL-derived path components with `isInvalid` before file system use.
5. Register the route in `New()` in `viewer.go`.
6. Add any new template defines to `templates/index.html` (or a new `*.html` file in `templates/`).
7. Add any new template helper functions to the FuncMap in `initTemplates` (`template.go`).

---

## Template data contracts

### `render_message`

Receives a `messageView` (defined in `handlers.go`), **not** a bare `slack.Message`.  
Use the `msgview` template func to construct one at the call site:

```
{{ template "render_message" (msgview $channelID $msg) }}
```

| Field | Type | Purpose |
|---|---|---|
| `.Msg` | `slack.Message` | The message to render |
| `.ChannelID` | `string` | Channel ID for reply-to anchor link; pass `""` to suppress the reply banner |

Pass `""` as `$channelID` in the thread panel (`hx_thread`), where the parent message lives on a different page and the anchor link would be broken.

---

## Template ARIA contracts

The tablist/tabpanel pair must satisfy:

| Element | Required attributes |
|---|---|
| Tab list container | `role="tablist"` |
| Each tab button | `role="tab"`, `aria-selected`, `aria-controls="{panel-id}"`, `tabindex` (roving) |
| Conversation panel | `id="conversation-panel"`, `role="tabpanel"`, `aria-labelledby="tab-conversation"`, `tabindex="0"` |
| Canvas panel | `id="canvas-panel"`, `role="tabpanel"`, `aria-labelledby="tab-canvas"`, `tabindex="0"` |

Keyboard navigation (Arrow Left/Right, Home, End) is implemented as inline JS within the
`tab_list` template define.
