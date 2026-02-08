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

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/processor"
)

func (cs *Stream) channelWorker(ctx context.Context, proc processor.Conversations, results chan<- Result, threadC chan<- request, reqs <-chan request) {
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

			// get the channel canvas
			if channel.Properties != nil && !channel.Properties.Canvas.IsEmpty {
				if err := cs.canvas(ctx, proc, channel, channel.Properties.Canvas.FileId); err != nil {
					// ignore canvas errors
					slog.Warn("canvas error", "err", err)
				}
			}

			if err := cs.channel(ctx, req, func(mm []slack.Message, isLast bool) error {
				n, err := procChanMsg(ctx, proc, threadC, channel, isLast, mm)
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

func (cs *Stream) threadWorker(ctx context.Context, proc processor.Conversations, results chan<- Result, threadReq <-chan request) {
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
			if !req.sl.IsThread() {
				results <- Result{Type: RTThread, Err: fmt.Errorf("invalid thread link: %s", req.sl)}
				continue
			}

			channel := new(slack.Channel)
			if req.threadOnly {
				var err error
				if channel, err = cs.procChannelInfoWithUsers(ctx, proc, req.sl.Channel, req.sl.ThreadTS); err != nil {
					results <- Result{Type: RTThread, ChannelID: req.sl.Channel, ThreadTS: req.sl.ThreadTS, Err: err}
					continue
				}
			} else {
				// hackety hack
				channel.ID = req.sl.Channel
			}
			if err := cs.thread(ctx, req, func(msgs []slack.Message, isLast bool) error {
				if err := procThreadMsg(ctx, proc, channel, req.sl.ThreadTS, req.threadOnly, isLast, msgs); err != nil {
					return err
				}
				results <- Result{Type: RTThread, ChannelID: req.sl.Channel, ThreadTS: req.sl.ThreadTS, IsLast: isLast}
				return nil
			}); err != nil {
				results <- Result{Type: RTThread, ChannelID: req.sl.Channel, ThreadTS: req.sl.ThreadTS, Err: err}
				continue
			}
		}
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

func (cs *Stream) canvas(ctx context.Context, proc processor.Conversations, channel *slack.Channel, fileId string) error {
	if fileId == "" {
		return nil
	}
	file, _, _, err := cs.client.GetFileInfoContext(ctx, fileId, 0, 1)
	if err != nil {
		return fmt.Errorf("canvas: %s: %w", fileId, err)
	}
	if file == nil {
		return errors.New("canvas: file not found")
	}
	if err := proc.Files(ctx, channel, slack.Message{}, []slack.File{*file}); err != nil {
		return fmt.Errorf("canvas: %s: %w", fileId, err)
	}
	return nil
}
