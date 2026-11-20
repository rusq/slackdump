// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"runtime/trace"
	"strings"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/client"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/progresslog"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/processor"
)

func (cs *Stream) progressOutcome(err error) error {
	if (errors.Is(err, errChanNotFound) || errors.Is(err, errNotInChannel)) && !cs.failChnlNotFnd {
		return nil
	}
	return err
}

func (cs *Stream) channelWorker(ctx context.Context, proc processor.Conversations, results chan<- Result, threadC chan<- ordinaryThreadRequest, canvasC chan<- canvasThreadRequest, canvasResultsC <-chan canvasThreadResult, reqs <-chan channelRequest) {
	ctx, task := trace.NewTask(ctx, "channelWorker")
	defer task.End()

	for {
		select {
		case <-ctx.Done():
			return
		case req, more := <-reqs:
			if !more {
				return // channel closed
			}
			stop := false
			func() {
				var outcome error
				defer func() { cs.progressEvent(ProgressChannelDone, req.sl.Channel, "", outcome) }()
				channel, err := cs.procChannelInfoWithUsers(ctx, proc, req.sl.Channel, req.sl.ThreadTS)
				if err != nil {
					outcome = cs.progressOutcome(err)
					if !sendResult(ctx, results, Result{Type: RTChannel, ChannelID: req.sl.Channel, Err: err}) {
						return
					}
					return
				}

				// Check for the channel canvas.
				if fileID, ok := structures.CanvasFileID(channel); ok {
					if err := cs.canvasFile(ctx, proc, channel, fileID); err != nil {
						if canvasErrorIsFatal(err) {
							outcome = err
							if !sendResult(ctx, results, Result{Type: RTChannel, ChannelID: req.sl.Channel, Err: err}) {
								return
							}
							return
						}
						logCanvasAPIError(ctx, "canvas file unavailable", channel.ID, fileID, "", "", err)
					}

					canvasClient, supported := cs.client.(client.CanvasRootClient)
					if !supported || !canvasClient.CanvasSupported() {
						slog.DebugContext(ctx, "skipping canvas discussions for non-client-token session", "owner_channel_id", channel.ID, "canvas_file_id", fileID)
					} else {
						if cm, ok := processor.AsCanvasMessenger(proc); ok {
							if err := cs.canvasDiscussions(ctx, proc, cm, canvasC, canvasResultsC, results, req, channel, fileID); err != nil {
								if canvasErrorIsFatal(err) {
									stop = true
									outcome = err
									return
								}
								hiddenID, _ := structures.CanvasChannelID(fileID)
								logCanvasAPIError(ctx, "canvas discussions unavailable", channel.ID, fileID, hiddenID, "", err)
							}
						}
					}
				}

				if err := cs.channel(ctx, req, func(mm []slack.Message, isLast bool) error {
					n, err := cs.procChanMsg(ctx, proc, threadC, channel, isLast, mm)
					if err != nil {
						return err
					}
					if !sendResult(ctx, results, Result{Type: RTChannel, ChannelID: req.sl.Channel, ThreadCount: n, IsLast: isLast}) {
						return context.Cause(ctx)
					}
					return nil
				}); err != nil {
					outcome = cs.progressOutcome(err)
					if !sendResult(ctx, results, Result{Type: RTChannel, ChannelID: req.sl.Channel, Err: err}) {
						return
					}
					return
				}
			}()
			if stop || ctx.Err() != nil {
				return
			}
		}
	}
}

func (cs *Stream) threadWorker(ctx context.Context, proc processor.Conversations, results chan<- Result, threadReq <-chan ordinaryThreadRequest) {
	ctx, task := trace.NewTask(ctx, "threadWorker")
	defer task.End()

	for {
		select {
		case <-ctx.Done():
			return
		case req, more := <-threadReq:
			if !more {
				return // channel closed
			}
			stop := false
			func() {
				var outcome error
				defer func() { cs.progressEvent(ProgressThreadDone, req.fetch.channelID, req.fetch.threadTS, outcome) }()
				if req.fetch.channelID == "" || req.fetch.threadTS == "" {
					outcome = errors.New("invalid ordinary thread request")
					if !sendResult(ctx, results, Result{Type: RTThread, ChannelID: req.fetch.channelID, ThreadTS: req.fetch.threadTS, Err: errors.New("invalid ordinary thread request")}) {
						stop = true
						return
					}
					return
				}

				channel := new(slack.Channel)
				if req.threadOnly {
					// Thread-only requests come from direct thread links (e.g., resume).
					// We only need channel info (ID, name, etc.) for file paths and
					// identification. Channel users are already recorded from the
					// original channel archive, and thread messages contain their own
					// user IDs. Skipping procChannelUsers saves an API call per thread.
					var err error
					if channel, err = cs.procChannelInfo(ctx, proc, req.fetch.channelID, req.fetch.threadTS); err != nil {
						outcome = cs.progressOutcome(err)
						if !sendResult(ctx, results, Result{Type: RTThread, ChannelID: req.fetch.channelID, ThreadTS: req.fetch.threadTS, Err: err}) {
							stop = true
							outcome = context.Cause(ctx)
							return
						}
						return
					}
				} else {
					// hackety hack
					// Threads discovered from channel messages. The channel info was
					// already fetched by channelWorker, so we just need the ID.
					channel.ID = req.fetch.channelID
				}
				firstPage := true
				err := cs.paginateThread(ctx, req.fetch, func(msgs []slack.Message, isLast bool) (bool, error) {
					if firstPage && req.threadOnly && cs.skipThread != nil && cs.skipThread(ctx, req.fetch.channelID, req.fetch.threadTS, msgs[0].ReplyCount) {
						slog.DebugContext(ctx, "skipping complete thread", "channel_id", req.fetch.channelID, "thread_ts", req.fetch.threadTS, "reply_count", msgs[0].ReplyCount)
						return false, nil
					}
					firstPage = false
					if err := procThreadMsg(ctx, proc, channel, req.fetch.threadTS, req.threadOnly, isLast, msgs); err != nil {
						return false, err
					}
					if !sendResult(ctx, results, Result{Type: RTThread, ChannelID: req.fetch.channelID, ThreadTS: req.fetch.threadTS, IsLast: isLast}) {
						stop = true
						return false, context.Cause(ctx)
					}
					return true, nil
				})
				if err != nil {
					if errors.Is(err, errThreadNotFound) {
						slog.WarnContext(cs.missingThreadLogContext(ctx, req.fetch.channelID), "skipping non-existing thread", "channel_id", cs.channelLogID(req.fetch.channelID), "thread_ts", req.fetch.threadTS)
						err = procThreadMsg(ctx, proc, channel, req.fetch.threadTS, req.threadOnly, true, parentOnlyThreadMessages(req.fetch, req.parent))
						if err == nil {
							if !sendResult(ctx, results, Result{Type: RTThread, ChannelID: req.fetch.channelID, ThreadTS: req.fetch.threadTS, IsLast: true}) {
								stop = true
								outcome = context.Cause(ctx)
								return
							}
							return
						}
					}
					outcome = cs.progressOutcome(err)
					if !sendResult(ctx, results, Result{Type: RTThread, ChannelID: req.fetch.channelID, ThreadTS: req.fetch.threadTS, Err: err}) {
						stop = true
						if outcome == nil {
							outcome = context.Cause(ctx)
						}
						return
					}
				}
			}()
			if stop {
				return
			}
		}
	}
}

// channelLogID renders a cached name in text logs while keeping JSON IDs as
// plain strings. It never performs a network lookup.
type namedChannelID struct{ id, name string }

func (v namedChannelID) String() string               { return v.id + " (#" + v.name + ")" }
func (v namedChannelID) MarshalJSON() ([]byte, error) { return json.Marshal(v.id) }

func (cs *Stream) cachedChannelName(channelID string) string {
	if channel := cs.chanCache.get(channelID); channel != nil {
		return strings.TrimLeft(strings.TrimSpace(progresslog.Sanitize(channel.Name)), "#")
	}
	return ""
}
func (cs *Stream) channelLogID(channelID string) any {
	if name := cs.cachedChannelName(channelID); name != "" {
		return namedChannelID{id: channelID, name: name}
	}
	return channelID
}

// missingThreadLogContext adds a progress-only description without fetching data.
func (cs *Stream) missingThreadLogContext(ctx context.Context, channelID string) context.Context {
	if cs.progress == nil {
		return ctx
	}
	label := channelID
	if name := cs.cachedChannelName(channelID); name != "" {
		label = "#" + name
	}
	return progresslog.WithMessage(ctx, "skipping non-existing thread in "+label)
}

func (cs *Stream) processCanvasThread(ctx context.Context, proc processor.Conversations, req canvasThreadRequest) error {
	if req.owner == nil || req.fetch.channelID == "" || req.fetch.threadTS == "" {
		return errors.New("invalid canvas thread request")
	}
	cm, ok := processor.AsCanvasMessenger(proc)
	if !ok {
		return errors.New("canvas processor capability is no longer available")
	}
	err := cs.paginateThread(ctx, req.fetch, func(msgs []slack.Message, isLast bool) (bool, error) {
		return true, procCanvasThreadMsg(ctx, proc, cm, req.owner, req.fetch.channelID, isLast, msgs)
	})
	if err == nil {
		return nil
	}
	if canvasErrorIsFatal(err) {
		return err
	}

	logCanvasAPIError(ctx, "canvas discussion unavailable", req.owner.ID, req.fileID, req.fetch.channelID, req.fetch.threadTS, err)
	if err := procCanvasThreadMsg(
		ctx,
		proc,
		cm,
		req.owner,
		req.fetch.channelID,
		true,
		parentOnlyThreadMessages(req.fetch, &req.root),
	); err != nil {
		return err
	}
	return nil
}

func (cs *Stream) canvasThreadWorker(ctx context.Context, proc processor.Conversations, completed chan<- canvasThreadResult, requests <-chan canvasThreadRequest) {
	ctx, task := trace.NewTask(ctx, "canvasThreadWorker")
	defer task.End()
	for req := range requests {
		result := canvasThreadResult{channelID: req.fetch.channelID, threadTS: req.fetch.threadTS}
		result.err = cs.processCanvasThread(ctx, proc, req)
		// Reserve the collector callback before marking the worker job done.
		cs.progressEvent(ProgressResultQueued, req.fetch.channelID, req.fetch.threadTS, nil)
		cs.progressEvent(ProgressThreadDone, req.fetch.channelID, req.fetch.threadTS, result.err)
		select {
		case completed <- result:
		case <-ctx.Done():
			return
		}
	}
}

type progressContextKey struct{}

func sendResult(ctx context.Context, results chan<- Result, res Result) bool {
	observer, _ := ctx.Value(progressContextKey{}).(func(ProgressEvent))
	tracked := res.Type == RTChannel || res.Type == RTThread || res.Type == RTCanvasThread
	if observer != nil && tracked {
		observer(ProgressEvent{Kind: ProgressResultQueued, ChannelID: res.ChannelID, ThreadTS: res.ThreadTS})
	}
	select {
	case results <- res:
		return true
	case <-ctx.Done():
		if observer != nil && tracked {
			observer(ProgressEvent{Kind: ProgressResultProcessed, ChannelID: res.ChannelID, ThreadTS: res.ThreadTS})
		}
		return false
	}
}

func (cs *Stream) channelInfoWorker(ctx context.Context, proc processor.ChannelInformer, srC chan<- Result, channelIdC <-chan string) {
	ctx, task := trace.NewTask(ctx, "channelInfoWorker")
	defer task.End()

	infoFetcher := cs.procChannelInfoWithUsers
	if cs.fastSearch {
		infoFetcher = cs.procChannelInfo
	}

	seen := make(map[string]struct{}, 512)

	for {
		select {
		case <-ctx.Done():
			return
		case id, more := <-channelIdC:
			if !more {
				return
			}
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}

			if _, err := infoFetcher(ctx, proc, id, ""); err != nil {
				// if _, err := cs.procChannelInfo(ctx, proc, id, ""); err != nil {
				if !sendResult(ctx, srC, Result{Type: RTChannelInfo, ChannelID: id, Err: fmt.Errorf("channelInfoWorker: %s: %w", id, err)}) {
					return
				}
			}
			seen[id] = struct{}{}
		}
	}
}

func (cs *Stream) canvasFile(ctx context.Context, proc processor.Conversations, channel *slack.Channel, fileID string) error {
	if fileID == "" {
		return nil
	}
	var file *slack.File
	if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier3.Retries, func(ctx context.Context) error {
		var err error
		file, _, _, err = cs.client.GetFileInfoContext(ctx, fileID, 0, 1)
		return err
	}, network.RetryReadOnly); err != nil {
		return newAPIError("files.info", err)
	}
	if file == nil {
		return newAPIError("files.info", errors.New("canvas file not found"))
	}
	if err := proc.Files(ctx, channel, slack.Message{}, []slack.File{*file}); err != nil {
		return fmt.Errorf("process canvas file %s: %w", fileID, err)
	}
	return nil
}

func (cs *Stream) canvasDiscussions(ctx context.Context, proc processor.Conversations, cm processor.CanvasMessenger, threadC chan<- canvasThreadRequest, completed <-chan canvasThreadResult, results chan<- Result, ownerReq channelRequest, owner *slack.Channel, fileID string) error {
	ctx, task := trace.NewTask(ctx, "canvasDiscussions")
	defer task.End()
	hiddenID, ok := structures.CanvasChannelID(fileID)
	if !ok {
		return newAPIError("canvas.threadRoots", fmt.Errorf("invalid canvas file ID %q", fileID))
	}
	canvasClient, ok := cs.client.(client.CanvasRootClient)
	if !ok || !canvasClient.CanvasSupported() {
		return newAPIError("canvas.threadRoots", client.ErrOpNotSupported)
	}
	var roots []slack.Message
	if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier3.Retries, func(ctx context.Context) error {
		var err error
		roots, err = canvasClient.CanvasThreadRoots(ctx, fileID)
		return err
	}); err != nil {
		err = newAPIError("canvas.threadRoots", err)
		if canvasErrorIsFatal(err) {
			sendResult(ctx, results, Result{Type: RTCanvasThread, ChannelID: hiddenID, Err: err})
		}
		return err
	}
	roots, err := cs.filterCanvasRoots(roots, ownerReq)
	if err != nil {
		return newAPIError("canvas.threadRoots", err)
	}
	requests, err := cs.procCanvasMsg(ctx, proc, cm, owner, hiddenID, fileID, ownerReq.oldest, ownerReq.latest, true, roots)
	if err != nil {
		if canvasErrorIsFatal(err) {
			sendResult(ctx, results, Result{Type: RTCanvasThread, ChannelID: hiddenID, Err: err})
		}
		return err
	}
	for _, r := range requests {
		cs.progressEvent(ProgressThreadQueued, r.fetch.channelID, r.fetch.threadTS, nil)
	}
	// Send and collect concurrently so neither bounded channel can deadlock on
	// a canvas with more discussions than its buffer.
	sent, received := 0, 0
	canvasResults := make([]canvasThreadResult, 0, len(requests))
	for received < len(requests) {
		var sendC chan<- canvasThreadRequest
		var next canvasThreadRequest
		if sent < len(requests) {
			sendC, next = threadC, requests[sent]
		}
		select {
		case sendC <- next:
			sent++
		case result := <-completed:
			canvasResults = append(canvasResults, result)
			received++
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	}
	var joined error
	firstFailure := -1
	for i, result := range canvasResults {
		if result.err == nil {
			if !sendResult(ctx, results, Result{Type: RTCanvasThread, ChannelID: result.channelID, ThreadTS: result.threadTS, IsLast: true}) {
				return context.Cause(ctx)
			}
			cs.progressEvent(ProgressResultProcessed, result.channelID, result.threadTS, nil)
			continue
		}
		cs.progressEvent(ProgressResultProcessed, result.channelID, result.threadTS, nil)
		if firstFailure < 0 {
			firstFailure = i
		}
		joined = errors.Join(joined, fmt.Errorf("canvas discussion channel=%s thread_ts=%s: %w", result.channelID, result.threadTS, result.err))
	}
	if firstFailure >= 0 {
		failed := canvasResults[firstFailure]
		if !sendResult(ctx, results, Result{Type: RTCanvasThread, ChannelID: failed.channelID, ThreadTS: failed.threadTS, IsLast: true, Err: joined}) {
			return context.Cause(ctx)
		}
		return joined
	}
	return nil
}

func (cs *Stream) filterCanvasRoots(messages []slack.Message, req channelRequest) ([]slack.Message, error) {
	oldest := structures.NVLTime(req.oldest, cs.oldest)
	latest := structures.NVLTime(req.latest, cs.latest)
	if oldest.IsZero() && latest.IsZero() {
		return messages, nil
	}
	filtered := make([]slack.Message, 0, len(messages))
	for _, message := range messages {
		ts, err := structures.ParseSlackTS(message.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("invalid canvas root timestamp %q: %w", message.Timestamp, err)
		}
		beforeOldest := !cs.includeOlderCanvasRoots &&
			!oldest.IsZero() &&
			(ts.Before(oldest) || (!cs.inclusive && ts.Equal(oldest)))
		afterLatest := !latest.IsZero() && (ts.After(latest) || (!cs.inclusive && ts.Equal(latest)))
		if !beforeOldest && !afterLatest {
			filtered = append(filtered, message)
		}
	}
	return filtered, nil
}

func (cs *Stream) procCanvasMsg(ctx context.Context, proc processor.Conversations, cm processor.CanvasMessenger, owner *slack.Channel, hiddenID, fileID string, oldest, latest time.Time, isLast bool, messages []slack.Message) ([]canvasThreadRequest, error) {
	roots := make([]slack.Message, 0, len(messages))
	threads := make([]canvasThreadRequest, 0, len(messages))
	for i := range messages {
		if messages[i].SubType != structures.SubTypeDocumentCommentRoot {
			continue
		}
		root := messages[i]
		if root.ThreadTimestamp == "" {
			root.ThreadTimestamp = root.Timestamp
		}
		roots = append(roots, root)
		if root.ReplyCount <= 0 {
			continue
		}
		if cs.skipCanvasThread != nil && cs.skipCanvasThread(ctx, hiddenID, root.ThreadTimestamp, root.ReplyCount) {
			slog.DebugContext(ctx, "skipping complete canvas discussion", "channel_id", hiddenID, "thread_ts", root.ThreadTimestamp, "reply_count", root.ReplyCount)
			cs.progressEvent(ProgressThreadQueued, hiddenID, root.ThreadTimestamp, nil)
			cs.progressEvent(ProgressThreadDone, hiddenID, root.ThreadTimestamp, nil)
			continue
		}
		parent := root
		threads = append(threads, canvasThreadRequest{
			fetch:  threadFetchSpec{channelID: hiddenID, threadTS: root.ThreadTimestamp, oldest: oldest, latest: latest},
			root:   parent,
			owner:  owner,
			fileID: fileID,
		})
	}

	if err := procFiles(ctx, proc, owner, roots...); err != nil {
		return nil, fmt.Errorf("process canvas root files: %w", err)
	}
	if err := cm.CanvasMessages(ctx, hiddenID, len(threads), isLast, roots); err != nil {
		return nil, fmt.Errorf("process canvas messages: %w", err)
	}
	return threads, nil
}

func procCanvasThreadMsg(ctx context.Context, proc processor.Conversations, cm processor.CanvasMessenger, owner *slack.Channel, hiddenChannelID string, isLast bool, messages []slack.Message) error {
	if len(messages) == 0 {
		return nil
	}
	parent := messages[0]
	if parent.ThreadTimestamp == "" {
		parent.ThreadTimestamp = parent.Timestamp
		messages[0] = parent
	}
	if err := procFiles(ctx, proc, owner, messages[1:]...); err != nil {
		return fmt.Errorf("process canvas discussion files: %w", err)
	}
	if err := cm.CanvasThreadMessages(ctx, hiddenChannelID, parent, isLast, messages); err != nil {
		return fmt.Errorf("process canvas discussion %s: %w", parent.ThreadTimestamp, err)
	}
	return nil
}

func canvasErrorIsFatal(err error) bool {
	if value, ok := errors.AsType[slack.SlackErrorResponse](err); ok && value.Err == "invalid_auth" {
		return true
	}
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		!isAPIError(err)
}

func logCanvasAPIError(ctx context.Context, message, ownerChannelID, fileID, hiddenChannelID, threadTS string, err error) {
	slog.WarnContext(ctx, message,
		"owner_channel_id", ownerChannelID,
		"canvas_file_id", fileID,
		"canvas_channel_id", hiddenChannelID,
		"thread_ts", threadTS,
		"err", err,
	)
}
