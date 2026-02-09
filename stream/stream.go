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
	"sync"
	"time"

	"github.com/rusq/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v4/internal/client"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/processor"
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

// Stream is used to fetch conversations from Slack.  It is safe for concurrent
// use.
type Stream struct {
	oldest, latest time.Time
	client         client.Slack
	limits         rateLimits
	chanCache      *chanCache
	fastSearch     bool
	inclusive      bool
	failChnlNotFnd bool // if true, will fail if channel not found
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
	RTChannelUsers
	RTSearch
)

// Result is sent to the callback function for each channel or thread.
type Result struct {
	Type        ResultType
	ChannelID   string
	ThreadTS    string
	ThreadCount int
	IsLast      bool // true if this is the last message for the channel or thread
	// Count contains the count of entities in the result. Right now it's
	// populated only for search results.
	Count int
	// Err contains the error if the result is an error.
	Err error
}

func (r Result) Error() string {
	return fmt.Sprintf("%s channel %s: %v", r.Type, structures.SlackLink{Channel: r.ChannelID, ThreadTS: r.ThreadTS}, r.Err)
}

func (r Result) Unwrap() error {
	return r.Err
}

func (r Result) String() string {
	switch r.Type {
	case RTSearch:
		return "<search>"
	default:
		if r.ThreadTS == "" {
			return "<" + r.ChannelID + ">"
		}
		return fmt.Sprintf("<%s[%s:%s]>", r.Type, r.ChannelID, r.ThreadTS)
	}
}

// rateLimits contains the rate limiters for the different tiers.
type rateLimits struct {
	channels    *rate.Limiter
	threads     *rate.Limiter
	users       *rate.Limiter
	userinfo    *rate.Limiter
	searchmsg   *rate.Limiter
	searchfiles *rate.Limiter
	tier        network.Limits
}

func limits(l network.Limits) rateLimits {
	return rateLimits{
		channels:    network.NewLimiter(network.Tier3, l.Tier3.Burst, int(l.Tier3.Boost)),
		threads:     network.NewLimiter(network.Tier3, l.Tier3.Burst, int(l.Tier3.Boost)),
		users:       network.NewLimiter(network.Tier2, l.Tier2.Burst, int(l.Tier2.Boost)),
		userinfo:    network.NewLimiter(network.Tier4, l.Tier4.Burst, int(l.Tier4.Boost)),
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

func OptFastSearch() Option {
	return func(cs *Stream) {
		cs.fastSearch = true
	}
}

func OptInclusive(b bool) Option {
	return func(cs *Stream) {
		cs.inclusive = b
	}
}

// OptFailOnChanNotFound enables or disables detection and special treatment of
// channel_not_found Slack errors.  If disabled, non-existing channels are
// skipped.
func OptFailOnChanNotFound(b bool) Option {
	return func(cs *Stream) {
		cs.failChnlNotFnd = b
	}
}

// New creates a new Stream instance that allows to stream different slack
// entities.
func New(cl client.Slack, l network.Limits, opts ...Option) *Stream {
	cs := &Stream{
		client:    cl,
		limits:    limits(l),
		chanCache: new(chanCache),
		inclusive: true,
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

// Users processes all users in the workspace, calling proc for each batch of
// users returned by the API.
func (cs *Stream) Users(ctx context.Context, proc processor.Users, opt ...slack.GetUsersOption) error {
	ctx, task := trace.NewTask(ctx, "Users")
	defer task.End()

	p := cs.client.GetUsersPaginated(opt...)
	var apiErr error
	for apiErr == nil {
		if apiErr = network.WithRetry(ctx, cs.limits.users, cs.limits.tier.Tier2.Retries, func(ctx context.Context) error {
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

// ListChannels calls processor for each batch of channels received from the API.
func (cs *Stream) ListChannels(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error {
	ctx, task := trace.NewTask(ctx, "Channels")
	defer task.End()

	var next string
	for {
		p.Cursor = next
		var ch []slack.Channel
		if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier3.Retries, func(ctx context.Context) error {
			var err error
			ch, next, err = cs.client.GetConversationsContext(ctx, p)
			return err
		}); err != nil {
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

// UsersBulk processes all users in the workspace, calling proc for each batch
// of users returned by the API.
func (cs *Stream) UsersBulk(ctx context.Context, proc processor.Users, ids ...string) error {
	ctx, task := trace.NewTask(ctx, "UsersBulk")
	defer task.End()

	uu := make([]slack.User, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			// some messages may have empty user IDs.
			continue
		}
		var u *slack.User
		if err := network.WithRetry(ctx, cs.limits.userinfo, cs.limits.tier.Tier4.Retries, func(ctx context.Context) error {
			var err error
			u, err = cs.client.GetUserInfoContext(ctx, id)
			return err
		}); err != nil {
			return fmt.Errorf("error fetching user with ID %s: %w", id, err)
		}
		uu = append(uu, *u)
	}
	if err := proc.Users(ctx, uu); err != nil {
		return err
	}
	return nil
}

// UsersBulkWithCustom returns the information for the users with ids and
// fetches custom profile fields.  If includeLabels is true, it will fetch the
// custom profile field names.
func (cs *Stream) UsersBulkWithCustom(ctx context.Context, proc processor.Users, includeLabels bool, ids ...string) error {
	ctx, task := trace.NewTask(ctx, "UsersBulkWithCustom")
	defer task.End()

	uu := make([]slack.User, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			// some messages may have empty user IDs.
			continue
		}

		var profileC = make(chan *slack.UserProfile, 1)
		go func() {
			defer close(profileC)
			var profile *slack.UserProfile
			// we ignore any errors, if we don't get extended information, we still have the basic from the GetUser call.
			err := network.WithRetry(ctx, cs.limits.userinfo, cs.limits.tier.Tier4.Retries, func(ctx context.Context) error {
				var err error
				profile, err = cs.client.GetUserProfileContext(ctx, &slack.GetUserProfileParameters{
					UserID:        id,
					IncludeLabels: includeLabels,
				})
				return err
			})
			if err != nil {
				slog.DebugContext(ctx, "profile fetch error", "error", err)
			}
			select {
			case profileC <- profile:
			case <-ctx.Done():
				return
			}
		}()

		var u *slack.User
		if err := network.WithRetry(ctx, cs.limits.userinfo, cs.limits.tier.Tier4.Retries, func(ctx context.Context) error {
			var err error
			u, err = cs.client.GetUserInfoContext(ctx, id)
			return err
		}); err != nil {
			<-profileC // discard
			return fmt.Errorf("error fetching user with ID %s: %w", id, err)
		}
		if profile := <-profileC; profile != nil {
			u.Profile = *profile
		}

		uu = append(uu, *u)
	}
	if err := proc.Users(ctx, uu); err != nil {
		return err
	}
	return nil
}
