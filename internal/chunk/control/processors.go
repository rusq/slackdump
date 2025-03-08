package control

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v3/internal/structures"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/processor"
)

// Special processors for the controller.

// userCollector collects users and sends the signal to start the transformer.
type userCollector struct {
	ctx   context.Context // bad boy, but short-lived, so it's ok
	users []slack.User
	ts    TransformStarter
}

var _ processor.Users = (*userCollector)(nil)

func (u *userCollector) Users(ctx context.Context, users []slack.User) error {
	u.users = append(u.users, users...)
	return nil
}

var errNoUsers = errors.New("no users collected")

// Close invokes the transformer's StartWithUsers method if it
// collected any users.
func (u *userCollector) Close() error {
	if len(u.users) == 0 {
		return errNoUsers
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
	finalised, err := ct.rc.IsFinalised(ctx, channelID)
	if err != nil {
		return fmt.Errorf("error checking if finalised: %w", err)
	}
	if !finalised {
		return nil
	}
	if err := ct.tf.Transform(ctx, chunk.ToFileID(channelID, threadID, threadOnly)); err != nil {
		return fmt.Errorf("error transforming: %w", err)
	}
	return nil
}

// chanFilter is a special processor that filters out channels based on the
// settings.  It also maintains an index of the channels that are in the list.
type chanFilter struct {
	links      chan<- structures.EntityItem
	list       *structures.EntityList
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
LOOP:
	for _, ch := range ch {
		if c.memberOnly && (ch.ID[0] == 'C' && !ch.IsMember) {
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
