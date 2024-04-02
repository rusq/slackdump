package stream

import (
	"context"
	"fmt"
	"runtime/trace"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/processor"
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
			channel, err := cs.channelInfoWithUsers(ctx, proc, req.sl.Channel, req.sl.ThreadTS)
			if err != nil {
				results <- Result{Type: RTChannel, ChannelID: req.sl.Channel, Err: err}
				continue
			}
			if err := cs.channel(ctx, req.sl.Channel, func(mm []slack.Message, isLast bool) error {
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

			var channel = new(slack.Channel)
			if req.threadOnly {
				var err error
				if channel, err = cs.channelInfoWithUsers(ctx, proc, req.sl.Channel, req.sl.ThreadTS); err != nil {
					results <- Result{Type: RTThread, ChannelID: req.sl.Channel, ThreadTS: req.sl.ThreadTS, Err: err}
					continue
				}
			} else {
				// hackety hack
				channel.ID = req.sl.Channel
			}
			if err := cs.thread(ctx, req.sl, func(msgs []slack.Message, isLast bool) error {
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

	var seen = make(map[string]struct{}, 512)

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
			if _, err := cs.channelInfo(ctx, proc, id, ""); err != nil {
				srC <- Result{Type: RTChannelInfo, ChannelID: id, Err: fmt.Errorf("channelInfoWorker: %s: %s", id, err)}
			}
			seen[id] = struct{}{}
		}
	}
}
