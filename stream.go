package slackdump

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"sync"
	"time"

	"github.com/rusq/slack"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/logger"
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

// Stream is used to fetch conversations from Slack.  It is safe for concurrent
// use.
type Stream struct {
	oldest, latest time.Time
	client         Slacker
	limits         rateLimits
	chanCache      *chanCache
	resultFn       []func(sr StreamResult) error
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
)

// StreamResult is sent to the callback function for each channel or thread.
type StreamResult struct {
	Type        ResultType // see below.
	ChannelID   string
	ThreadTS    string
	ThreadCount int
	IsLast      bool // true if this is the last message for the channel or thread
	Err         error
}

func (s StreamResult) String() string {
	if s.ThreadTS == "" {
		return "<" + s.ChannelID + ">"
	}
	return fmt.Sprintf("<%s[%s:%s]>", s.Type, s.ChannelID, s.ThreadTS)
}

// rateLimits contains the rate limiters for the different tiers.
type rateLimits struct {
	channels *rate.Limiter
	threads  *rate.Limiter
	users    *rate.Limiter
	tier     *Limits
}

func limits(l *Limits) rateLimits {
	return rateLimits{
		channels: network.NewLimiter(network.Tier3, l.Tier3.Burst, int(l.Tier3.Boost)),
		threads:  network.NewLimiter(network.Tier3, l.Tier3.Burst, int(l.Tier3.Boost)),
		users:    network.NewLimiter(network.Tier2, l.Tier2.Burst, int(l.Tier2.Boost)),
		tier:     l,
	}
}

// StreamOption functions are used to configure the stream.
type StreamOption func(*Stream)

// OptOldest sets the oldest time to be fetched.
func OptOldest(t time.Time) StreamOption {
	return func(cs *Stream) {
		cs.oldest = t
	}
}

// OptLatest sets the latest time to be fetched.
func OptLatest(t time.Time) StreamOption {
	return func(cs *Stream) {
		cs.latest = t
	}
}

// OptResultFn sets the callback function that is called for each result.
func OptResultFn(fn func(sr StreamResult) error) StreamOption {
	return func(cs *Stream) {
		cs.resultFn = append(cs.resultFn, fn)
	}
}

// NewStream creates a new Stream instance that allows to stream different
// slack entities.
func NewStream(cl Slacker, l *Limits, opts ...StreamOption) *Stream {
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

// SyncConversations fetches the conversations from the link which can be a
// channelID, channel URL, thread URL or a link in Slackdump format.
func (cs *Stream) SyncConversations(ctx context.Context, proc processor.Conversations, link ...string) error {
	lg := logger.FromContext(ctx)
	return cs.ConversationsCB(ctx, proc, link, func(sr StreamResult) error {
		lg.Debugf("stream: finished processing: %s", sr)
		return nil
	})
}

func (cs *Stream) ConversationsCB(ctx context.Context, proc processor.Conversations, link []string, cb func(StreamResult) error) error {
	ctx, task := trace.NewTask(ctx, "channelStream.Conversations")
	defer task.End()

	lg := logger.FromContext(ctx)
	cs.resultFn = append(cs.resultFn, cb)

	linkC := make(chan string, 1)
	go func() {
		defer close(linkC)
		for _, l := range link {
			linkC <- l
		}
		lg.Debugf("stream: sent %d links", len(link))
	}()

	if err := cs.Conversations(ctx, proc, linkC); err != nil {
		return err
	}
	return nil
}

// Conversations fetches the conversations from the links channel.  The link
// sent on that channel can be a channelID, channel URL, thread URL or a link
// in Slackdump format.  fn is called for each result (channel messages, or
// thread messages).  The fact that fn was called for channel messages, does
// not mean that all threads for that channel were already processed.  Each
// last thread result is marked with StreamResult.IsLast.  The caller must
// track the number of threads processed for each channel, and when the thread
// result with IsLast is received, the caller can assume that all threads and
// messages for that channel have been processed.  For example, see
// [cmd/slackdump/internal/export/expproc].
func (cs *Stream) Conversations(ctx context.Context, proc processor.Conversations, links <-chan string) error {
	ctx, task := trace.NewTask(ctx, "AsyncConversations")
	defer task.End()

	// create channels
	chansC := make(chan request, msgChanSz)
	threadsC := make(chan request, threadChanSz)

	resultsC := make(chan StreamResult, resultSz)

	var wg sync.WaitGroup
	{
		// channel worker
		wg.Add(1)
		go func() {
			defer wg.Done()
			cs.channelWorker(ctx, proc, resultsC, threadsC, chansC)
			// we close threads here, instead of the main loop, because we want to
			// close it after all the threads are sent by channels.
			close(threadsC)
			trace.Log(ctx, "async", "channel worker done")
		}()
	}
	{
		// thread worker
		wg.Add(1)
		go func() {
			defer wg.Done()
			cs.threadWorker(ctx, proc, resultsC, threadsC)
			trace.Log(ctx, "async", "thread worker done")
		}()
	}
	{
		// main loop
		wg.Add(1)
		go func() {
			defer trace.Log(ctx, "async", "main loop done")
			defer wg.Done()
			defer close(chansC)
			for {
				select {
				case <-ctx.Done():
					resultsC <- StreamResult{Type: RTMain, Err: ctx.Err()}
					return
				case link, more := <-links:
					if !more {
						return
					}
					if err := processLink(chansC, threadsC, link); err != nil {
						resultsC <- StreamResult{Type: RTMain, Err: fmt.Errorf("link error: %q: %w", link, err)}
					}
				}
			}
		}()
	}
	go func() {
		// sentinel waits for all the workers to finish, then closes the error
		// channel.
		wg.Wait()
		close(resultsC)
		trace.Log(ctx, "async", "sentinel done")
	}()

	// result processing.
	for res := range resultsC {
		if err := res.Err; err != nil {
			trace.Logf(ctx, "error", "type: %s, chan_id: %s, thread_ts: %s, error: %s", res.Type, res.ChannelID, res.ThreadTS, err.Error())
			return err
		}
		for _, fn := range cs.resultFn {
			if err := fn(res); err != nil {
				return err
			}
		}
	}
	trace.Log(ctx, "info", "complete")
	return nil
}

// processLink parses the link and sends it to the appropriate output channel.
func processLink(chans chan<- request, threads chan<- request, link string) error {
	sl, err := structures.ParseLink(link)
	if err != nil {
		return err
	}
	if !sl.IsValid() {
		return fmt.Errorf("invalid slack link: %s", link)
	}
	if sl.IsThread() {
		threads <- request{sl: &sl, threadOnly: true}
	} else {
		chans <- request{sl: &sl}
	}
	return nil
}

type request struct {
	sl *structures.SlackLink
	// threadOnly indicates that this is the thread directly requested by the
	// user, and not a thread that was found in the channel.
	threadOnly bool
}

func (we *StreamResult) Error() string {
	return fmt.Sprintf("%s channel %s: %v", we.Type, structures.SlackLink{Channel: we.ChannelID, ThreadTS: we.ThreadTS}, we.Err)
}

func (we *StreamResult) Unwrap() error {
	return we.Err
}

func (cs *Stream) channelWorker(ctx context.Context, proc processor.Conversations, results chan<- StreamResult, threadC chan<- request, reqs <-chan request) {
	ctx, task := trace.NewTask(ctx, "channelWorker")
	defer task.End()

	for {
		select {
		case <-ctx.Done():
			results <- StreamResult{Type: RTChannel, Err: ctx.Err()}
			return
		case req, more := <-reqs:
			if !more {
				return // channel closed
			}
			channel, err := cs.channelInfoWithUsers(ctx, proc, req.sl.Channel, req.sl.ThreadTS)
			if err != nil {
				results <- StreamResult{Type: RTChannel, ChannelID: req.sl.Channel, Err: err}
				continue
			}
			if err := cs.channel(ctx, req.sl.Channel, func(mm []slack.Message, isLast bool) error {
				n, err := procChanMsg(ctx, proc, threadC, channel, isLast, mm)
				if err != nil {
					return err
				}
				results <- StreamResult{Type: RTChannel, ChannelID: req.sl.Channel, ThreadCount: n, IsLast: isLast}
				return nil
			}); err != nil {
				results <- StreamResult{Type: RTChannel, ChannelID: req.sl.Channel, Err: err}
				continue
			}
		}
	}
}

func (cs *Stream) channel(ctx context.Context, id string, callback func(mm []slack.Message, isLast bool) error) error {
	ctx, task := trace.NewTask(ctx, "channel")
	defer task.End()

	lg := logger.FromContext(ctx)

	cursor := ""
	for {
		var resp *slack.GetConversationHistoryResponse
		if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier3.Retries, func() error {
			var apiErr error
			r := trace.StartRegion(ctx, "GetConversationHistoryContext")
			defer r.End()
			resp, apiErr = cs.client.GetConversationHistoryContext(ctx, &slack.GetConversationHistoryParameters{
				ChannelID: id,
				Cursor:    cursor,
				Limit:     cs.limits.tier.Request.Conversations,
				Oldest:    structures.FormatSlackTS(cs.oldest),
				Latest:    structures.FormatSlackTS(cs.latest),
				Inclusive: true,
			})
			return apiErr
		}); err != nil {
			return err
		}
		if !resp.Ok {
			trace.Logf(ctx, "error", "not ok, api error=%s", resp.Error)
			return fmt.Errorf("response not ok, slack error: %s", resp.Error)
		}

		r := trace.StartRegion(ctx, "channel_callback")
		err := callback(resp.Messages, !resp.HasMore)
		r.End()
		if err != nil {
			// lg.Printf("channel %s, callback error: %s", id, err)
			return fmt.Errorf("channel %s, callback error: %w", id, err)
		}

		if !resp.HasMore {
			lg.Debugf("server reported channel %s done", id)
			break
		}
		cursor = resp.ResponseMetaData.NextCursor
	}
	return nil
}

func (cs *Stream) threadWorker(ctx context.Context, proc processor.Conversations, results chan<- StreamResult, threadReq <-chan request) {
	ctx, task := trace.NewTask(ctx, "threadWorker")
	defer task.End()

	for {
		select {
		case <-ctx.Done():
			results <- StreamResult{Type: RTThread, Err: ctx.Err()}
			return
		case req, more := <-threadReq:
			if !more {
				return // channel closed
			}
			if !req.sl.IsThread() {
				results <- StreamResult{Type: RTThread, Err: fmt.Errorf("invalid thread link: %s", req.sl)}
				continue
			}

			var channel = new(slack.Channel)
			if req.threadOnly {
				var err error
				if channel, err = cs.channelInfoWithUsers(ctx, proc, req.sl.Channel, req.sl.ThreadTS); err != nil {
					results <- StreamResult{Type: RTThread, ChannelID: req.sl.Channel, ThreadTS: req.sl.ThreadTS, Err: err}
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
				results <- StreamResult{Type: RTThread, ChannelID: req.sl.Channel, ThreadTS: req.sl.ThreadTS, IsLast: isLast}
				return nil
			}); err != nil {
				results <- StreamResult{Type: RTThread, ChannelID: req.sl.Channel, ThreadTS: req.sl.ThreadTS, Err: err}
				continue
			}
		}
	}
}

// thread fetches the whole thread identified by SlackLink, calling callback
// function fn for each slice received.
func (cs *Stream) thread(ctx context.Context, sl *structures.SlackLink, callback func(mm []slack.Message, isLast bool) error) error {
	ctx, task := trace.NewTask(ctx, "thread")
	defer task.End()

	if !sl.IsThread() {
		return fmt.Errorf("not a thread: %s", sl)
	}

	lg := logger.FromContext(ctx)
	lg.Debugf("- getting: %s", sl)

	var cursor string
	for {
		var (
			msgs    []slack.Message
			hasmore bool
		)
		if err := network.WithRetry(ctx, cs.limits.threads, cs.limits.tier.Tier3.Retries, func() error {
			var apiErr error
			msgs, hasmore, cursor, apiErr = cs.client.GetConversationRepliesContext(ctx, &slack.GetConversationRepliesParameters{
				ChannelID: sl.Channel,
				Timestamp: sl.ThreadTS,
				Cursor:    cursor,
				Limit:     cs.limits.tier.Request.Replies,
				Oldest:    structures.FormatSlackTS(cs.oldest),
				Latest:    structures.FormatSlackTS(cs.latest),
				Inclusive: true,
			})
			return apiErr
		}); err != nil {
			return err
		}

		// got just the leader message, no replies
		if len(msgs) <= 1 {
			return nil
		}

		r := trace.StartRegion(ctx, "thread_callback")
		err := callback(msgs, !hasmore)
		r.End()
		if err != nil {
			return err
		}

		if !hasmore {
			break
		}
	}
	return nil
}

// procChanMsg processes the messages in the channel and sends
// thread requests for the threads in the channel, if it discovers messages
// with threads.  It returns thread count in the mm and error if any.
func procChanMsg(ctx context.Context, proc processor.Conversations, threadC chan<- request, channel *slack.Channel, isLast bool, mm []slack.Message) (int, error) {
	lg := logger.FromContext(ctx)

	var trs = make([]request, 0, len(mm))
	for i := range mm {
		// collecting threads to get their count.  But we don't start
		// processing them yet, before we send the messages with the number of
		// "expected" threads to processor, to ensure that processor will
		// start processing the channel and will have the initial reference
		// count, if it needs it.
		if mm[i].Msg.ThreadTimestamp != "" && mm[i].Msg.SubType != "thread_broadcast" && mm[i].LatestReply != structures.NoRepliesLatestReply {
			lg.Debugf("- message #%d/channel=%s,thread: id=%s, thread_ts=%s", i, channel.ID, mm[i].Timestamp, mm[i].Msg.ThreadTimestamp)
			trs = append(trs, request{
				sl: &structures.SlackLink{
					Channel:  channel.ID,
					ThreadTS: mm[i].Msg.ThreadTimestamp,
				},
			})
		}
		if err := procFiles(ctx, proc, channel, mm[i]); err != nil {
			return len(trs), err
		}
	}
	if err := proc.Messages(ctx, channel.ID, len(trs), isLast, mm); err != nil {
		return 0, fmt.Errorf("failed to process message chunk starting with id=%s (size=%d): %w", mm[0].Msg.Timestamp, len(mm), err)
	}
	for _, tr := range trs {
		threadC <- tr
	}
	return len(trs), nil
}

func procThreadMsg(ctx context.Context, proc processor.Conversations, channel *slack.Channel, threadTS string, threadOnly bool, isLast bool, msgs []slack.Message) error {
	// extract files from thread messages
	if len(msgs) == 0 {
		return errors.New("empty messages slice")
	}
	if err := procFiles(ctx, proc, channel, msgs[1:]...); err != nil {
		return err
	}
	// slack returns the thread starter as the first message with every
	// call, so we use it as a parent message.
	if err := proc.ThreadMessages(ctx, channel.ID, msgs[0], threadOnly, isLast, msgs[1:]); err != nil {
		return fmt.Errorf("failed to process thread message id=%s, thread_ts=%s: %w", msgs[0].Msg.Timestamp, threadTS, err)
	}
	return nil
}

func procFiles(ctx context.Context, proc processor.Filer, channel *slack.Channel, msgs ...slack.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	for _, m := range msgs {
		if len(m.Files) > 0 {
			if err := proc.Files(ctx, channel, m, m.Files); err != nil {
				return err
			}
		}
	}
	return nil
}

// channelInfo fetches the channel info and passes it to the processor.
func (cs *Stream) channelInfo(ctx context.Context, proc processor.ChannelInformer, channelID string, threadTS string) (*slack.Channel, error) {
	ctx, task := trace.NewTask(ctx, "channelInfo")
	defer task.End()

	trace.Logf(ctx, "channel_id", "%s, threadTS=%q", channelID, threadTS)

	// to avoid fetching the same channel info multiple times, we cache it.
	var info *slack.Channel
	if info = cs.chanCache.get(channelID); info == nil {
		if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier3.Retries, func() error {
			var err error
			info, err = cs.client.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
				ChannelID:         channelID,
				IncludeLocale:     true,
				IncludeNumMembers: true,
			})
			if err != nil {
				return fmt.Errorf("error getting channel information: %w", err)
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("api error: %s: %w", channelID, err)
		}
		cs.chanCache.set(channelID, info)
	}
	if err := proc.ChannelInfo(ctx, info, threadTS); err != nil {
		return nil, err
	}

	return info, nil
}

func (cs *Stream) channelUsers(ctx context.Context, proc processor.ChannelInformer, channelID, threadTS string) ([]string, error) {
	var uu []string
	var cursor string
	for {
		var u []string
		var next string
		if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier4.Retries, func() error {
			var err error
			u, next, err = cs.client.GetUsersInConversationContext(ctx, &slack.GetUsersInConversationParameters{
				ChannelID: channelID,
				Cursor:    cursor,
			})
			return err
		}); err != nil {
			return nil, fmt.Errorf("error getting conversation users: %w", err)
		}
		if err := proc.ChannelUsers(ctx, channelID, threadTS, u); err != nil {
			return nil, err
		}
		uu = append(uu, u...)
		if next == "" {
			break
		}
		cursor = next
	}
	return uu, nil
}

// channelInfoWithUsers returns the slack channel with members populated from
// another api.
func (cs *Stream) channelInfoWithUsers(ctx context.Context, proc processor.ChannelInformer, channelID, threadTS string) (*slack.Channel, error) {
	var eg errgroup.Group

	var chC = make(chan slack.Channel, 1)
	eg.Go(func() error {
		defer close(chC)
		ch, err := cs.channelInfo(ctx, proc, channelID, threadTS)
		if err != nil {
			return err
		}
		chC <- *ch
		return nil
	})

	var uC = make(chan []string, 1)
	eg.Go(func() error {
		defer close(uC)
		m, err := cs.channelUsers(ctx, proc, channelID, threadTS)
		if err != nil {
			return err
		}
		uC <- m
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	ch := <-chC
	ch.Members = <-uC
	return &ch, nil
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
