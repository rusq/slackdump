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
	"errors"
	"fmt"
	"log/slog"
	"runtime/trace"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/client"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/processor"
)

func (cs *Stream) channelWorker(ctx context.Context, proc processor.Conversations, results chan<- Result, threadC chan<- ordinaryThreadRequest, canvasC chan<- canvasThreadRequest, canvasResultsC <-chan canvasThreadResult, reqs <-chan channelRequest) {
	ctx, task := trace.NewTask(ctx, "channelWorker")
	defer task.End()

	for {
		select {
		case <-ctx.Done():
			results <- Result{Type: RTChannel, Err: ctx.Err()}
			return
		case req, more := <-reqs:
			if !more {
				return // channel closed
			}
			channel, err := cs.procChannelInfoWithUsers(ctx, proc, req.sl.Channel, req.sl.ThreadTS)
			if err != nil {
				results <- Result{Type: RTChannel, ChannelID: req.sl.Channel, Err: err}
				continue
			}

			// Check for the channel canvas.
			if fileID, ok := structures.CanvasFileID(channel); ok {
				if err := cs.canvasFile(ctx, proc, channel, fileID); err != nil {
					if canvasErrorIsFatal(err) {
						results <- Result{Type: RTChannel, ChannelID: req.sl.Channel, Err: err}
						continue
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
				results <- Result{Type: RTChannel, ChannelID: req.sl.Channel, ThreadCount: n, IsLast: isLast}
				return nil
			}); err != nil {
				results <- Result{Type: RTChannel, ChannelID: req.sl.Channel, Err: err}
				continue
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
			results <- Result{Type: RTThread, Err: ctx.Err()}
			return
		case req, more := <-threadReq:
			if !more {
				return // channel closed
			}
			if req.fetch.channelID == "" || req.fetch.threadTS == "" {
				results <- Result{Type: RTThread, ChannelID: req.fetch.channelID, ThreadTS: req.fetch.threadTS, Err: errors.New("invalid ordinary thread request")}
				continue
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
					results <- Result{Type: RTThread, ChannelID: req.fetch.channelID, ThreadTS: req.fetch.threadTS, Err: err}
					continue
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
				results <- Result{Type: RTThread, ChannelID: req.fetch.channelID, ThreadTS: req.fetch.threadTS, IsLast: isLast}
				return true, nil
			})
			if err != nil {
				if errors.Is(err, errThreadNotFound) {
					slog.WarnContext(ctx, "skipping non-existing thread", "channel_id", req.fetch.channelID, "thread_ts", req.fetch.threadTS)
					err = procThreadMsg(ctx, proc, channel, req.fetch.threadTS, req.threadOnly, true, parentOnlyThreadMessages(req.fetch, req.parent))
					if err == nil {
						results <- Result{Type: RTThread, ChannelID: req.fetch.channelID, ThreadTS: req.fetch.threadTS, IsLast: true}
						continue
					}
				}
				results <- Result{Type: RTThread, ChannelID: req.fetch.channelID, ThreadTS: req.fetch.threadTS, Err: err}
			}
		}
	}
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
		completed <- result
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
				srC <- Result{Type: RTChannelInfo, ChannelID: id, Err: fmt.Errorf("channelInfoWorker: %s: %w", id, err)}
			}
			seen[id] = struct{}{}
		}
	}
}

func (cs *Stream) canvasFile(ctx context.Context, proc processor.Conversations, channel *slack.Channel, fileID string) error {
	if fileID == "" {
		return nil
	}
	file, _, _, err := cs.client.GetFileInfoContext(ctx, fileID, 0, 1)
	if err != nil {
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
		return newAPIError("canvas.threadRoots", err)
	}
	roots, err := cs.filterCanvasRoots(roots, ownerReq)
	if err != nil {
		return newAPIError("canvas.threadRoots", err)
	}
	requests, err := cs.procCanvasMsg(ctx, proc, cm, owner, hiddenID, fileID, ownerReq.oldest, ownerReq.latest, true, roots)
	if err != nil {
		return err
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
			results <- Result{Type: RTCanvasThread, ChannelID: result.channelID, ThreadTS: result.threadTS, IsLast: true}
			continue
		}
		if firstFailure < 0 {
			firstFailure = i
		}
		joined = errors.Join(joined, fmt.Errorf("canvas discussion channel=%s thread_ts=%s: %w", result.channelID, result.threadTS, result.err))
	}
	if firstFailure >= 0 {
		failed := canvasResults[firstFailure]
		results <- Result{Type: RTCanvasThread, ChannelID: failed.channelID, ThreadTS: failed.threadTS, IsLast: true, Err: joined}
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
