package stream

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"sync"
	"time"

	"github.com/rusq/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/rusq/slackdump/v3/processor"
)

const (
	// message channel buffer size.  Messages are much faster than threads, so
	// we can have a smaller buffer.
	msgChanSz = 16
	// thread channel buffer size.  Threads are much slower than channels,
	// because each message might have a thread, and that means, that we'll
	// have to send a thread request for each message.  So, we need a larger
	// buffer for it not to block the channel messages scraping.
	threadChanSz = 4000
	// result channel buffer size.  We are running 2 goroutines, 1 for channel
	// messages, and 1 for threads.
	resultSz = 2
)

// Slacker is the interface with some functions of slack.Client.
type Slacker interface {
	AuthTestContext(context.Context) (response *slack.AuthTestResponse, err error)
	GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error)
	GetUsersPaginated(options ...slack.GetUsersOption) slack.UserPagination

	GetStarredContext(ctx context.Context, params slack.StarsParameters) ([]slack.StarredItem, *slack.Paging, error)
	ListBookmarks(channelID string) ([]slack.Bookmark, error)

	GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error)
	GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error)
	GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error)

	SearchMessagesContext(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchMessages, error)
	SearchFilesContext(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchFiles, error)
}

// Stream is used to fetch conversations from Slack.  It is safe for concurrent
// use.
type Stream struct {
	oldest, latest time.Time
	client         Slacker
	limits         rateLimits
	chanCache      *chanCache
	resultFn       []func(sr Result) error
}

// chanCache is used to cache channel info to avoid fetching it multiple times.
type chanCache struct {
	m sync.Map
}

// get returns the channel info from the cache.  If it fails to find it, it
// returns nil.
func (c *chanCache) get(key string) *slack.Channel {
	v, ok := c.m.Load(key)
	if !ok {
		return nil
	}
	return v.(*slack.Channel)
}

// set sets the channel info in the cache under the respective key.
func (c *chanCache) set(key string, ch *slack.Channel) {
	c.m.Store(key, ch)
}

// ResultType helps to identify the type of the result, so that the callback
// function can handle it appropriately.
//
//go:generate stringer -type=ResultType -trimprefix=RT
type ResultType int8

const (
	RTMain    ResultType = iota // Main function result
	RTChannel                   // Result containing channel information
	RTThread                    // Result containing thread information
	RTChannelInfo
)

// Result is sent to the callback function for each channel or thread.
type Result struct {
	Type        ResultType
	ChannelID   string
	ThreadTS    string
	ThreadCount int
	IsLast      bool // true if this is the last message for the channel or thread
	Err         error
}

func (s Result) String() string {
	if s.ThreadTS == "" {
		return "<" + s.ChannelID + ">"
	}
	return fmt.Sprintf("<%s[%s:%s]>", s.Type, s.ChannelID, s.ThreadTS)
}

// rateLimits contains the rate limiters for the different tiers.
type rateLimits struct {
	channels    *rate.Limiter
	threads     *rate.Limiter
	users       *rate.Limiter
	searchmsg   *rate.Limiter
	searchfiles *rate.Limiter
	tier        *network.Limits
}

func limits(l *network.Limits) rateLimits {
	return rateLimits{
		channels:    network.NewLimiter(network.Tier3, l.Tier3.Burst, int(l.Tier3.Boost)),
		threads:     network.NewLimiter(network.Tier3, l.Tier3.Burst, int(l.Tier3.Boost)),
		users:       network.NewLimiter(network.Tier2, l.Tier2.Burst, int(l.Tier2.Boost)),
		searchmsg:   network.NewLimiter(network.Tier2, l.Tier2.Burst, int(l.Tier2.Boost)),
		searchfiles: network.NewLimiter(network.Tier2, l.Tier2.Burst, int(l.Tier2.Boost)),
		tier:        l,
	}
}

// Option functions are used to configure the stream.
type Option func(*Stream)

// OptOldest sets the oldest time to be fetched.
func OptOldest(t time.Time) Option {
	return func(cs *Stream) {
		cs.oldest = t
	}
}

// OptLatest sets the latest time to be fetched.
func OptLatest(t time.Time) Option {
	return func(cs *Stream) {
		cs.latest = t
	}
}

// OptResultFn sets the callback function that is called for each result.
func OptResultFn(fn func(sr Result) error) Option {
	return func(cs *Stream) {
		cs.resultFn = append(cs.resultFn, fn)
	}
}

// New creates a new Stream instance that allows to stream different
// slack entities.
func New(cl Slacker, l *network.Limits, opts ...Option) *Stream {
	cs := &Stream{
		client:    cl,
		limits:    limits(l),
		chanCache: new(chanCache),
	}
	for _, opt := range opts {
		opt(cs)
	}
	if cs.oldest.After(cs.latest) {
		cs.oldest, cs.latest = cs.latest, cs.oldest
	}
	return cs
}

// WorkspaceInfo fetches the workspace info and passes it to the processor.
// Getting it might be needed when the transformer need the current User ID or
// Team ID. (Different teams within one workspace are not yet supported.)
func (cs *Stream) WorkspaceInfo(ctx context.Context, proc processor.WorkspaceInfo) error {
	ctx, task := trace.NewTask(ctx, "WorkspaceInfo")
	defer task.End()

	atr, err := cs.client.AuthTestContext(ctx)
	if err != nil {
		return err
	}

	return proc.WorkspaceInfo(ctx, atr)
}

// Users returns all users in the workspace.
func (cs *Stream) Users(ctx context.Context, proc processor.Users, opt ...slack.GetUsersOption) error {
	ctx, task := trace.NewTask(ctx, "Users")
	defer task.End()

	p := cs.client.GetUsersPaginated(opt...)
	var apiErr error
	for apiErr == nil {
		if apiErr = network.WithRetry(ctx, cs.limits.users, cs.limits.tier.Tier2.Retries, func() error {
			var err error
			p, err = p.Next(ctx)
			return err
		}); apiErr != nil {
			break
		}
		if err := proc.Users(ctx, p.Users); err != nil {
			return err
		}
	}

	return p.Failure(errors.Unwrap(apiErr))
}

// TODO: test this.
func (cs *Stream) ListChannels(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error {
	ctx, task := trace.NewTask(ctx, "Channels")
	defer task.End()

	var next string
	for {
		p.Cursor = next
		var (
			ch  []slack.Channel
			err error
		)
		ch, next, err = cs.client.GetConversationsContext(ctx, p)
		if err != nil {
			return fmt.Errorf("API error: %w", err)
		}

		// this can happen if we're running the stream under the guest user.
		// slack returns empty chunks.
		if len(ch) == 0 {
			if next == "" {
				break
			}
			continue
		}
		if err := proc.Channels(ctx, ch); err != nil {
			return err
		}
		if next == "" {
			break
		}
	}
	return nil
}
