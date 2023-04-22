package slackdump

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"sync"
	"time"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/internal/chunk/processor"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

const (
	msgChanSz    = 16   // message channel buffer size
	threadChanSz = 2000 // thread channel buffer size
	resultSz     = 2    // result channel buffer size
)

type Stream struct {
	oldest, latest time.Time
	client         streamer
	limits         rateLimits
}

type StreamResult struct {
	Type        ResultType // "channel" or "thread"
	ChannelID   string
	ThreadTS    string
	ThreadCount int
	IsLast      bool // true if this is the last message for the channel or thread
	Err         error
}

//go:generate stringer -type=ResultType -trimprefix=RT
type ResultType int8

const (
	RTMain ResultType = iota
	RTChannel
	RTThread
)

func (s StreamResult) String() string {
	if s.ThreadTS == "" {
		return "<" + s.ChannelID + ">"
	}
	return fmt.Sprintf("<%s[%s:%s]>", s.Type, s.ChannelID, s.ThreadTS)
}

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

func NewStream(cl streamer, l *Limits, opts ...StreamOption) *Stream {
	return newChannelStream(cl, l, opts...)
}

func newChannelStream(cl streamer, l *Limits, opts ...StreamOption) *Stream {
	cs := &Stream{
		client: cl,
		limits: limits(l),
	}
	for _, opt := range opts {
		opt(cs)
	}
	if cs.oldest.After(cs.latest) {
		cs.oldest, cs.latest = cs.latest, cs.oldest
	}
	return cs
}

// Conversations fetches the conversations from the link which can be a
// channelID, channel URL, thread URL or a link in Slackdump format.
func (cs *Stream) Conversations(ctx context.Context, proc processor.Conversations, link ...string) error {
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

	linkC := make(chan string, 1)
	go func() {
		defer close(linkC)
		for _, l := range link {
			linkC <- l
		}
		lg.Debugf("stream: sent %d links", len(link))
	}()

	if err := cs.AsyncConversations(ctx, proc, linkC, cb); err != nil {
		return err
	}
	return nil
}

// AsyncConversations fetches the conversations from the link which can be a
// channelID, channel URL, thread URL or a link in Slackdump format.  fn is
// called for each result (channel messages, or thread messages).  The fact
// that fn was called for channel messages, does not mean that all threads for
// that channel have been processed.  The fn is called for each thread result,
// and the last thread result is marked with StreamResult.IsLast.  The caller
// must track the number of threads processed for each channel, and when the
// thread result with IsLast is received, the caller can assume that all
// threads and messages for that channel have been processed.  For example,
// see [cmd/slackdump/internal/export/expproc].
func (cs *Stream) AsyncConversations(ctx context.Context, proc processor.Conversations, links <-chan string, fn func(StreamResult) error) error {
	ctx, task := trace.NewTask(ctx, "AsyncConversations")
	defer task.End()

	// create channels
	chansC := make(chan channelRequest, msgChanSz)
	threadsC := make(chan threadRequest, threadChanSz)

	resultsC := make(chan StreamResult, resultSz)

	var wg sync.WaitGroup
	{
		// channel worker
		wg.Add(1)
		go func() {
			cs.channelWorker(ctx, proc, resultsC, threadsC, chansC)
			// we close threads here, instead of the main loop, because we want to
			// close it after all the thread workers are done.
			close(threadsC)
			wg.Done()
			trace.Log(ctx, "async", "channel worker done")
		}()
	}
	{
		// thread worker
		wg.Add(1)
		go func() {
			cs.threadWorker(ctx, proc, resultsC, threadsC)
			wg.Done()
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
					if err := cs.processLink(chansC, threadsC, link); err != nil {
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

	for res := range resultsC {
		if err := res.Err; err != nil {
			trace.Log(ctx, "error", err.Error())
			return err
		}
		if err := fn(res); err != nil {
			return err
		}
	}
	trace.Log(ctx, "func", "complete")
	return nil
}

// processLink parses the link and sends it to the appropriate worker.
func (cs *Stream) processLink(chans chan<- channelRequest, threads chan<- threadRequest, link string) error {
	sl, err := structures.ParseLink(link)
	if err != nil {
		return err
	}
	if !sl.IsValid() {
		return fmt.Errorf("invalid slack link: %s", link)
	}
	if sl.IsThread() {
		threads <- threadRequest{channelID: sl.Channel, threadTS: sl.ThreadTS, needChanInfo: true}
	} else {
		chans <- channelRequest{channelID: sl.Channel}
	}
	return nil
}

type channelRequest struct {
	channelID string
}

type threadRequest struct {
	channelID string
	threadTS  string
	// needChanInfo indicates whether the channel info is needed for the thread.
	// This is true when we're fetching the standalone thread without the
	// conversation.
	needChanInfo bool
}

func (we *StreamResult) Error() string {
	return fmt.Sprintf("%s channel %s: %v", we.Type, structures.SlackLink{Channel: we.ChannelID, ThreadTS: we.ThreadTS}, we.Err)
}

func (we *StreamResult) Unwrap() error {
	return we.Err
}

func (cs *Stream) channelWorker(ctx context.Context, proc processor.Conversations, results chan<- StreamResult, threadC chan<- threadRequest, reqs <-chan channelRequest) {
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
			if err := cs.channelInfo(ctx, proc, req.channelID, false); err != nil {
				results <- StreamResult{Type: RTChannel, ChannelID: req.channelID, Err: err}
			}
			last := false
			threadCount := 0
			if err := cs.channel(ctx, req.channelID, func(mm []slack.Message, isLast bool) error {
				last = isLast
				n, err := processChannelMessages(ctx, proc, threadC, req.channelID, isLast, mm)
				threadCount = n
				return err
			}); err != nil {
				results <- StreamResult{Type: RTChannel, ChannelID: req.channelID, Err: err}
			}
			results <- StreamResult{Type: RTChannel, ChannelID: req.channelID, ThreadCount: threadCount, IsLast: last}
		}
	}
}

func (cs *Stream) channel(ctx context.Context, id string, fn func(mm []slack.Message, isLast bool) error) error {
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
		err := fn(resp.Messages, !resp.HasMore)
		r.End()
		if err != nil {
			lg.Printf("channel %s, callback error: %s", id, err)
			return fmt.Errorf("callback error: %w", err)
		}

		if !resp.HasMore {
			lg.Debugf("server reported channel %s done", id)
			break
		}
		cursor = resp.ResponseMetaData.NextCursor
	}
	return nil
}

func (cs *Stream) threadWorker(ctx context.Context, proc processor.Conversations, results chan<- StreamResult, reqs <-chan threadRequest) {
	ctx, task := trace.NewTask(ctx, "threadWorker")
	defer task.End()

	for {
		select {
		case <-ctx.Done():
			results <- StreamResult{Type: RTThread, Err: ctx.Err()}
			return
		case req, more := <-reqs:
			if !more {
				return // channel closed
			}
			if req.needChanInfo {
				if err := cs.channelInfo(ctx, proc, req.channelID, true); err != nil {
					results <- StreamResult{Type: RTThread, ChannelID: req.channelID, ThreadTS: req.threadTS, Err: err}
				}
			}
			var last bool
			if err := cs.thread(ctx, req.channelID, req.threadTS, func(msgs []slack.Message, isLast bool) error {
				last = isLast
				return processThreadMessages(ctx, proc, req.channelID, req.threadTS, isLast, msgs)
			}); err != nil {
				results <- StreamResult{Type: RTThread, ChannelID: req.channelID, ThreadTS: req.threadTS, Err: err}
			}
			results <- StreamResult{Type: RTThread, ChannelID: req.channelID, ThreadTS: req.threadTS, IsLast: last}
		}
	}
}

func (cs *Stream) thread(ctx context.Context, id string, threadTS string, fn func(mm []slack.Message, isLast bool) error) error {
	ctx, task := trace.NewTask(ctx, "thread")
	defer task.End()

	lg := logger.FromContext(ctx)
	lg.Debugf("- getting: thread: id=%s, thread_ts=%s", id, threadTS)

	var cursor string
	for {
		var (
			msgs    []slack.Message
			hasmore bool
		)
		if err := network.WithRetry(ctx, cs.limits.threads, cs.limits.tier.Tier3.Retries, func() error {
			var apiErr error
			msgs, hasmore, cursor, apiErr = cs.client.GetConversationRepliesContext(ctx, &slack.GetConversationRepliesParameters{
				ChannelID: id,
				Timestamp: threadTS,
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
		err := fn(msgs, !hasmore)
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

// processChannelMessages processes the messages in the channel and sends
// thread requests for the threads in the channel, if it discovers messages
// with threads.  It returns thread count in the mm and error if any.
func processChannelMessages(ctx context.Context, proc processor.Conversations, threadC chan<- threadRequest, channelID string, isLast bool, mm []slack.Message) (int, error) {
	lg := logger.FromContext(ctx)

	var trs = make([]threadRequest, 0, len(mm))
	for i := range mm {
		// collecting threads to get their count.  But we don't start
		// processing them yet, before we send the messages with the number of
		// "expected" threads to processor, to ensure that processor will
		// start processing the channel and will have the initial reference
		// count, if it needs it.
		if mm[i].Msg.ThreadTimestamp != "" && mm[i].Msg.SubType != "thread_broadcast" && mm[i].LatestReply != structures.NoRepliesLatestReply {
			lg.Debugf("- message #%d/channel=%s,thread: id=%s, thread_ts=%s", i, channelID, mm[i].Timestamp, mm[i].Msg.ThreadTimestamp)
			trs = append(trs, threadRequest{channelID: channelID, threadTS: mm[i].Msg.ThreadTimestamp})
		}
		if len(mm[i].Files) > 0 {
			if err := proc.Files(ctx, channelID, mm[i], false, mm[i].Files); err != nil {
				return len(trs), err
			}
		}
	}
	if err := proc.Messages(ctx, channelID, len(trs), isLast, mm); err != nil {
		return 0, fmt.Errorf("failed to process message chunk starting with id=%s (size=%d): %w", mm[0].Msg.Timestamp, len(mm), err)
	}
	for _, tr := range trs {
		threadC <- tr
	}
	return len(trs), nil
}

func processThreadMessages(ctx context.Context, proc processor.Conversations, channelID, threadTS string, isLast bool, msgs []slack.Message) error {
	// extract files from thread messages
	for _, m := range msgs[1:] {
		if len(m.Files) > 0 {
			if err := proc.Files(ctx, channelID, m, true, m.Files); err != nil {
				return err
			}
		}
	}
	// slack returns the thread starter as the first message with every
	// call, so we use it as a parent message.
	if err := proc.ThreadMessages(ctx, channelID, msgs[0], isLast, msgs[1:]); err != nil {
		return fmt.Errorf("failed to process thread message id=%s, thread_ts=%s: %w", msgs[0].Msg.Timestamp, threadTS, err)
	}
	return nil
}

// channelInfo fetches the channel info and passes it to the processor.
func (cs *Stream) channelInfo(ctx context.Context, proc processor.Conversations, channelID string, isThread bool) error {
	ctx, task := trace.NewTask(ctx, "channelInfo")
	defer task.End()

	trace.Logf(ctx, "channel_id", "%s, isThread=%v", channelID, isThread)

	var info *slack.Channel
	if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier3.Retries, func() error {
		var err error
		info, err = cs.client.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
			ChannelID: channelID,
		})
		return err
	}); err != nil {
		return err
	}
	if err := proc.ChannelInfo(ctx, info, isThread); err != nil {
		return err
	}
	return nil
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
		var (
			ch  []slack.Channel
			err error
		)
		p.Cursor = next
		ch, next, err = cs.client.GetConversationsContext(ctx, p)
		if err != nil {
			return err
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
