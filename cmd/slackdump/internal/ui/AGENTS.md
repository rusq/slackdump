# TUI Agent Notes

Reusable notes for Bubble Tea TUI work in this repository.

## Config UI

- `cfgui.Model` expects focus before it handles key messages in `Update`; tests that exercise interaction should call `SetFocus(true)`.
- Numeric shortcuts in config screens are one-based row selectors. Always bound-check the zero-based target against `m.last` before moving the cursor.
- Boolean config parameters use `updaters.BoolModel` as immediate toggles. In `cfgui`, run the updater's `Init()` message through `Update()` locally, keep the model in `selecting`, avoid assigning `m.child`, and queue a config refresh.
- Non-boolean config updaters should continue through the child editor path: inline params set `state = inline`, other editable params set `state = editing`, and both mount `m.child`.
- `WMClose` handling is the normal refresh path for child editors. Avoid changing it when adding direct-toggle behavior.
- For table rendering, keep display values stable with fixed padding and `nvl` for empty values. Use disabled value styling only for params without an updater.

## Focused Tests

- Test model behavior directly with `tea.KeyMsg` values instead of relying on terminal snapshots when the assertion is about state, cursor movement, or mounted children.
- For number shortcuts, include both a valid row and an out-of-range row case.
- For boolean config interactions, assert the backing bool changed, `state` stayed `selecting`, and `child` stayed nil.
- For non-boolean interactions, keep coverage that editable inline and modal params still mount their updater child.

## Shared Keymap and Help Baseline

- Use `ui.DefaultHuhKeymap()` for Huh forms/fields instead of calling `huh.NewDefaultKeyMap()` locally. It carries the repository help-label overrides and returns a fresh instance per call (huh forms mutate keymap state, so instances must not be shared between live forms).
- When adding Huh select-like fields, verify both behavior and displayed help labels. The shared Huh keymap should advertise `↑/k` and `↓/j` for Select, MultiSelect, and FilePicker navigation.
- Use `ui.NewHelp()` for Bubble help models instead of raw `help.New()` so custom controls inherit `ui.DefaultTheme().Help`.
- Prefer the shared key-label constants and binding helpers from `cmd/slackdump/internal/ui/keymap.go` for custom Bubble controls. Keep accepted shortcut keys unchanged unless the task explicitly asks for behavior changes.
- If help text changes intentionally, update snapshot-style view tests in the same change. `filemgr` view tests are sensitive to exact help strings.
