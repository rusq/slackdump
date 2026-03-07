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

package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/rusq/slackdump/v4/internal/structures"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/processor"
)

// Special processors for the controller.

// userCollector collects users and sends the signal to start the transformer.
type userCollector struct {
	ctx        context.Context // bad boy, but short-lived, so it's ok
	users      []slack.User
	ts         TransformStarter
	allowEmpty bool
}

var _ processor.Users = (*userCollector)(nil)

func (u *userCollector) Users(ctx context.Context, users []slack.User) error {
	u.users = append(u.users, users...)
	return nil
}

var ErrNoUsers = errors.New("no users returned")

// Close invokes the transformer's StartWithUsers method if it
// collected any users.
func (u *userCollector) Close() error {
	if len(u.users) == 0 {
		if u.allowEmpty {
			slog.Warn("user collector: no users collected, possibly not an error")
		} else {
			return ErrNoUsers
		}
	}
	if err := u.ts.StartWithUsers(u.ctx, u.users); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("error starting the transformer: %w", err)
	}
	return nil
}

// conversationTransformer monitors the "last" calls, and once
// the reference count drops to zero, it starts the transformer for
// the channel.  It should be appended after the main conversation
// processor.
type conversationTransformer struct {
	ctx context.Context
	tf  chunk.Transformer
	rc  ReferenceChecker
}

var _ processor.Messenger = (*conversationTransformer)(nil)

func (ct *conversationTransformer) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, mm []slack.Message) error {
	if isLast {
		return ct.mbeTransform(ctx, channelID, "", false)
	}
	return nil
}

func (ct *conversationTransformer) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly bool, isLast bool, tm []slack.Message) error {
	if isLast {
		return ct.mbeTransform(ctx, channelID, parent.ThreadTimestamp, threadOnly)
	}
	return nil
}

func (ct *conversationTransformer) mbeTransform(ctx context.Context, channelID, threadID string, threadOnly bool) error {
	// there are two cases:
	// 1. we are in a thread-only mode, so we need to check if this thread individual thread is complete.
	if threadOnly {
		return ct.mbeTransformThread(ctx, channelID, threadID)
	}
	// 2. we are in a channel mode, so we need to check if the channel is complete.
	return ct.mbeTransformChannel(ctx, channelID)
}

func (ct *conversationTransformer) mbeTransformChannel(ctx context.Context, channelID string) error {
	isComplete, err := ct.rc.IsComplete(ctx, channelID)
	if err != nil {
		return fmt.Errorf("error checking if complete: %w", err)
	}
	lg := slog.With("channel_id", channelID, "is_complete", isComplete)
	lg.Debug("channel finalisation")
	if !isComplete {
		lg.Debug("not complete, skipping")
		return nil
	}
	lg.Debug("calling channel transform")
	if err := ct.tf.Transform(ctx, channelID, ""); err != nil {
		return fmt.Errorf("error transforming: %w", err)
	}
	return nil
}

func (ct *conversationTransformer) mbeTransformThread(ctx context.Context, channelID string, threadID string) error {
	isComplete, err := ct.rc.IsCompleteThread(ctx, channelID, threadID)
	if err != nil {
		return fmt.Errorf("error checking if complete: %w", err)
	}
	lg := slog.With("channel_id", channelID, "thread_id", threadID, "is_complete", isComplete, "thread_only", true)
	lg.Debug("thread finalisation")
	if !isComplete {
		lg.Debug("not complete, skipping")
		return nil
	}
	lg.Debug("calling thread transform")
	// TODO: TransformThread #511
	if err := ct.tf.Transform(ctx, channelID, threadID); err != nil {
		return fmt.Errorf("error transforming: %w", err)
	}
	return nil
}

// chanFilter is a special processor that filters out channels based on the
// settings.  It also maintains an index of the channels that are in the list.
type chanFilter struct {
	links      chan<- structures.EntityItem
	memberOnly bool
	idx        map[string]*structures.EntityItem
}

// newChanFilter creates a new channel filter.
func newChanFilter(links chan<- structures.EntityItem, list *structures.EntityList, memberOnly bool) *chanFilter {
	return &chanFilter{
		links:      links,
		memberOnly: memberOnly,
		idx:        list.Index(),
	}
}

var _ processor.Channels = (*chanFilter)(nil)

// Channels called by Stream, scans the channel list ch and if the
// channel matches the filter, and is not excluded or duplicate, it sends the
// channel ID (as an EntityItem) to the links channel.
func (c *chanFilter) Channels(ctx context.Context, ch []slack.Channel) error {
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
	}
LOOP:
	for _, ch := range ch {
		if c.memberOnly && !structures.IsMember(&ch) {
			// skip public non-member channels
			continue
		}
		for _, entry := range c.idx {
			if !entry.Include && entry.Id == ch.ID {
				// skip excluded items
				continue LOOP
			}
		}
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case c.links <- structures.EntityItem{Id: ch.ID, Include: true}:
		}
	}
	return nil
}

// combinedChannels filters the processed channels and outputs previously not seen ones.
type combinedChannels struct {
	output    chan<- structures.EntityItem
	processed map[string]struct{}
}

var _ processor.Channels = (*combinedChannels)(nil)

func (c *combinedChannels) Channels(ctx context.Context, ch []slack.Channel) error {
	for _, ch := range ch {
		if _, ok := c.processed[ch.ID]; ok {
			continue
		}
		c.processed[ch.ID] = struct{}{}
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case c.output <- structures.EntityItem{Id: ch.ID, Include: true}:
		}
	}
	return nil
}

// errEmitter returns a function that sends the error to the error channel.
func errEmitter(errC chan<- error, sub string, stage Stage) func(err error) {
	return func(err error) {
		errC <- Error{
			Subroutine: sub,
			Stage:      stage,
			Err:        err,
		}
	}
}

// jointFileSearcher allows to customize the file processor on file search.
type jointFileSearcher struct {
	processor.FileSearcher
	filer processor.Filer
}

// Files method override.
func (j *jointFileSearcher) Files(ctx context.Context, ch *slack.Channel, msg slack.Message, files []slack.File) error {
	return j.filer.Files(ctx, ch, msg, files)
}

// Close method override.
func (j *jointFileSearcher) Close() error {
	var errs error
	if err := j.FileSearcher.Close(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("error closing file searcher: %w", err))
	}
	if err := j.filer.Close(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("error closing file processor: %w", err))
	}
	return errs
}

type msgUserIDsCollector struct {
	seen    map[string]struct{}
	userIDC chan []string
}

var _ processor.Messenger = (*msgUserIDsCollector)(nil)

func newMsgUserIDsCollector() *msgUserIDsCollector {
	const prealloc = 100
	userIDC := make(chan []string, prealloc)
	return &msgUserIDsCollector{
		seen:    make(map[string]struct{}, prealloc),
		userIDC: userIDC,
	}
}

func (uic *msgUserIDsCollector) Close() error {
	close(uic.userIDC)
	return nil
}

func (uic *msgUserIDsCollector) C() <-chan []string {
	return uic.userIDC
}

func (uic *msgUserIDsCollector) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, mm []slack.Message) error {
	return uic.collect(ctx, mm)
}

func (uic *msgUserIDsCollector) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly bool, isLast bool, tm []slack.Message) error {
	return uic.collect(ctx, tm)
}

func (uic *msgUserIDsCollector) collect(ctx context.Context, mm []slack.Message) error {
	var uu []string
	for _, m := range mm {
		user := m.User
		if user == "" {
			// TODO: support bot IDs, i.e. m.SubType == "bot_message" and m.BotID holding ID.
			// this would require adding GetBotInfo method to the Streamer and Slacker interfaces.
			continue
		}
		if _, ok := uic.seen[user]; ok {
			continue
		}
		uic.seen[user] = struct{}{}
		uu = append(uu, user)
	}
	if len(uu) > 0 {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case uic.userIDC <- uu:
		}
	}

	return nil
}
