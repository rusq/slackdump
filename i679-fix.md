# Fix Plan: `resume -threads -skip-complete-threads`

## Summary

Current `master` fixes PR #679's original bypass by dropping database thread items
whenever `-skip-complete-threads` is set. That makes the skip predicate run for
threads discovered while scanning channel history, but it can miss new replies on
old thread parents outside the channel `-lookback` window.

The fix is to include direct resume thread items again when `-threads` is enabled
and apply the same "complete thread" skip decision in the direct thread fetch path.

## Key Changes

- In `cmd/slackdump/internal/resume/resume.go`, change `latest(...)` so thread
  items are skipped only when `includeThreads` is false. With
  `includeThreads=true`, include thread items regardless of `skipCompleteThreads`.
- Remove or reword the runtime warning and flag/help text that says old thread
  parents outside lookback are not checked, because direct thread items will cover
  that case after this fix.
- In `stream.thread`, apply `cs.skipThread` for `req.threadOnly` requests after
  the first successful `conversations.replies` response and before calling the
  processor callback.
- Use a dedicated `firstPage` boolean for the direct-thread skip check. Do not
  rely on `cursor == ""` after the API call, because the call updates `cursor`.
- Use the first returned message's `ReplyCount` as Slack's current reply count.
  If `cs.skipThread(ctx, channelID, threadTS, replyCount)` returns true, log the
  skip and return nil without invoking the callback.
- Keep the existing `procChanMsg` skip behavior for channel-discovered threads.
  Do not apply the new direct-thread skip to those requests, because their parent
  was already counted in the channel result's expected thread count.

## API And Behavior

- No new CLI flags, public interfaces, or database schema changes.
- `-skip-complete-threads` still cannot detect edits, deletes, or reaction-only
  changes. It only compares stored message count with Slack `reply_count + 1`.
- For direct historical thread items, one `conversations.replies` call is still
  required to read Slack's current `reply_count`. Complete threads avoid full
  pagination and database writes, not every API call.
- Existing resume overlap and duplicate handling remains unchanged.

## Test Plan

- Update `Test_latest`: with `includeThreads=true` and
  `skipCompleteThreads=true`, thread items should be included in the returned
  entity list.
- Extend `TestStream_thread` with a direct-thread complete case: the first
  replies page contains a parent with `ReplyCount`, `skipThread` returns true,
  and the callback is not called.
- Extend `TestStream_thread` with a direct-thread incomplete case: `skipThread`
  returns false, and the callback receives the already-fetched first page.
- Keep existing `Test_procChanMsg` coverage for channel-discovered skip behavior.
- Run:

```bash
go test ./stream ./cmd/slackdump/internal/resume ./internal/chunk/backend/dbase
```

## Assumptions

- Success means `resume -threads -skip-complete-threads` must not miss new replies
  in historical threads solely because their parent message is outside
  `-lookback`.
- The accepted performance tradeoff is one lightweight replies call per direct
  historical thread to get Slack's current `reply_count`.
