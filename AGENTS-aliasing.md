# Aliasing Branch Review

Review target: committed branch delta from `master...aliasing`

## Findings

1. The write paths do not write anything.

   - `internal/viewer/handlers.go`: `aliasPutHandler` and `aliasDeleteHandler` render templates but do not read submitted alias data and do not call `SetAlias` or `DeleteAlias`.
   - Current behavior can report visual success without persisting any alias state.

2. The new UI is unreachable from the normal conversation page.

   - `internal/viewer/templates/index.html`: `hx_chan_header` contains the alias button.
   - The actual conversation page still renders `conversation_header`, which does not include `hx_chan_header`.
   - Result: users do not get an alias entry point in the main flow.

3. Alias state is never loaded into display rendering.

   - `mainView.Alias` and `mainView.CanAlias` were added, but `Alias` is never populated and is not used by templates.
   - Channel names still come from `rendername -> UserIndex.ChannelName`.
   - Even with persistence added, aliases would not appear in the rendered UI without a shared display-name path.

4. Capability checks are inconsistent.

   - `PUT /archives/{id}/alias/` checks `v.canAlias()`.
   - `GET /archives/{id}/alias/` and `DELETE /archives/{id}/alias/` do not.
   - Unsupported sources can therefore expose a misleading alias UI path.

## Design Questions

Resolved:

1. Aliases persist in the underlying `source.Sourcer` when it implements `Aliaser`, following the same general capability pattern as `GetFileByID`.
2. Only the database-backed source should implement alias persistence.
3. Storage model is a SQLite `ALIAS` table:

   - `CHANNEL_ID TEXT NOT NULL`
   - `ALIAS TEXT`
   - `CREATED_AT TIMESTAMP DEFAULT CURRENT_TIMESTAMP`
   - `CHANNEL_ID` is the primary key
   - `CHANNEL_ID` has a foreign key to `CHANNELS.ID`
   - insert on first add
   - update `ALIAS.ALIAS` for the same `CHANNEL_ID` when the user changes an existing alias
   - hard delete on delete

4. SQL repository implementation should live in `internal/chunk/backend/dbase/repository/alias.go`.

5. Aliases replace the rendered channel name everywhere when set.
6. Aliased names should render in italic to distinguish them from canonical names/IDs.
7. All conversation types support aliases.
8. Alias validation rules:

   - trim surrounding spaces before validation
   - empty result means delete
   - maximum length: 30 characters
   - allowed characters only:
     - Unicode letters
     - Unicode digits
     - underscore `_`
     - dash `-`

## Execution Plan

1. Add SQLite migration and repository support for the `ALIAS` table in the database-backed source, with SQL implementation in `internal/chunk/backend/dbase/repository/alias.go`.
2. Expose alias operations through the database-backed `source.Sourcer` implementation via `Aliaser`.
3. Add a single shared channel display-name path so aliases affect sidebar, headers, conversation titles, and related links consistently.
4. Render aliased names in italics wherever they replace canonical names.
5. Wire the HTMX alias flow into the real conversation header rather than a disconnected partial.
6. Implement request handling fully: parse form data, trim input, apply validation, treat empty as delete, insert on first add, update on subsequent edits for the same channel, and return correct error codes.
7. Add tests for repository persistence, source integration, validation behavior, handler behavior, and rendered alias display.
