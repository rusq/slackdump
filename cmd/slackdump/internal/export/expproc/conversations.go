package expproc

import (
	"context"
	"runtime/trace"
	"sync"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2/internal/chunk/processor"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/slack-go/slack"
)

// Conversations is a processor that writes the channel and thread messages.
type Conversations struct {
	dir   string
	cw    map[string]*channelproc
	mu    sync.RWMutex
	lg    logger.Interface
	filer processor.Filer

	// flags
	recordFiles bool

	onFinalise func(ctx context.Context, channelID string) error
}

// ConvOption is a function that configures the Conversations processor.
type ConvOption func(*Conversations)

// FinaliseFunc sets a callback function that is called when the processor is
// finished processing all channel and threads for the channel (when the
// reference count becomes 0).
func FinaliseFunc(fn func(ctx context.Context, channelID string) error) ConvOption {
	return func(cv *Conversations) {
		cv.onFinalise = fn
	}
}

// WithLogger sets the logger for the processor.
func WithLogger(lg logger.Interface) ConvOption {
	return func(cv *Conversations) {
		cv.lg = lg
	}
}

// WithRecordFiles sets whether the files should be recorded in the chunk file.
func WithRecordFiles(b bool) ConvOption {
	return func(cv *Conversations) {
		cv.recordFiles = b
	}
}

type channelproc struct {
	*baseproc
	// refcnt is the number of threads are expected to be processed for
	// the given channel.  We keep track of the number of threads, to ensure
	// that we don't close the file until all threads are processed.
	// The channel file can be closed when the number of threads is zero.
	refcnt int
}

// NewConversation returns the new conversation processor.  filer.Files method
// will be called for each file chunk.
func NewConversation(dir string, filer processor.Filer, opts ...ConvOption) (*Conversations, error) {
	c := &Conversations{
		dir:   dir,
		lg:    logger.Default,
		cw:    make(map[string]*channelproc),
		filer: filer,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// ensure ensures that the channel file is open and the recorder is
// initialized.
func (cv *Conversations) ensure(channelID string) error {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	if _, ok := cv.cw[channelID]; ok {
		return nil
	}
	bp, err := newBaseProc(cv.dir, channelID)
	if err != nil {
		return err
	}
	cv.cw[channelID] = &channelproc{
		baseproc: bp,
		refcnt:   1, // the channel itself is a reference
	}
	return nil
}

// ChannelInfo is called for each channel that is retrieved.
func (cv *Conversations) ChannelInfo(ctx context.Context, ci *slack.Channel, isThread bool) error {
	r, err := cv.recorder(ci.ID)
	if err != nil {
		return err
	}
	return r.ChannelInfo(ctx, ci, isThread)
}

func (cv *Conversations) recorder(channelID string) (*baseproc, error) {
	r, ok := cv.cw[channelID]
	if ok {
		return r.baseproc, nil
	}
	if err := cv.ensure(channelID); err != nil {
		return nil, err
	}
	cv.mu.RLock()
	defer cv.mu.RUnlock()
	return cv.cw[channelID].baseproc, nil
}

// refcount returns the number of references that are expected to be
// processed for the given channel.
func (cv *Conversations) refcount(channelID string) int {
	cv.mu.RLock()
	defer cv.mu.RUnlock()
	if _, ok := cv.cw[channelID]; !ok {
		return 0
	}
	return cv.cw[channelID].refcnt
}

func (cv *Conversations) incRefN(channelID string, n int) {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	if _, ok := cv.cw[channelID]; !ok {
		return
	}
	cv.cw[channelID].refcnt += n
}

func (cv *Conversations) decRef(channelID string) {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	if _, ok := cv.cw[channelID]; !ok {
		return
	}
	cv.cw[channelID].refcnt--
}

// Messages is called for each message that is retrieved.
func (cv *Conversations) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, mm []slack.Message) error {
	ctx, task := trace.NewTask(ctx, "Messages")
	defer task.End()

	lg := dlog.FromContext(ctx)
	lg.Debugf("processor: channelID=%s, numThreads=%d, isLast=%t, len(mm)=%d", channelID, numThreads, isLast, len(mm))
	r, err := cv.recorder(channelID)
	if err != nil {
		return err
	}
	if numThreads > 0 {
		cv.incRefN(channelID, numThreads) // one for each thread
		trace.Logf(ctx, "ref", "added %d", numThreads)
		lg.Debugf("processor: increased ref count for %q to %d", channelID, cv.refcount(channelID))
	}
	if err := r.Messages(ctx, channelID, numThreads, isLast, mm); err != nil {
		return err
	}
	if isLast {
		trace.Log(ctx, "isLast", "true, decrease ref count")
		cv.decRef(channelID)
		return cv.finalise(ctx, channelID)
	}
	return nil
}

// Files is called for each file that is retrieved. The parent message is
// passed in as well.
func (cv *Conversations) Files(ctx context.Context, channelID string, parent slack.Message, isThread bool, ff []slack.File) error {
	if err := cv.filer.Files(ctx, channelID, parent, isThread, ff); err != nil {
		return err
	}
	if cv.recordFiles {
		r, err := cv.recorder(channelID)
		if err != nil {
			return err
		}
		if err := r.Files(ctx, channelID, parent, isThread, ff); err != nil {
			return err
		}
	}
	return nil
}

// ThreadMessages is called for each of the thread messages that are
// retrieved. The parent message is passed in as well.
func (cv *Conversations) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, isLast bool, tm []slack.Message) error {
	ctx, task := trace.NewTask(ctx, "ThreadMessages")
	defer task.End()

	r, err := cv.recorder(channelID)
	if err != nil {
		return err
	}
	if err := r.ThreadMessages(ctx, channelID, parent, isLast, tm); err != nil {
		return err
	}
	cv.decRef(channelID)
	trace.Logf(ctx, "ref", "decremented, current=%d", cv.refcount(channelID))
	if isLast {
		trace.Log(ctx, "isLast", "true")
		return cv.finalise(ctx, channelID)
	}
	return nil
}

// finalise closes the channel file if there are no more threads to process.
func (cv *Conversations) finalise(ctx context.Context, channelID string) error {
	lg := dlog.FromContext(ctx)
	if tc := cv.refcount(channelID); tc > 0 {
		trace.Logf(ctx, "ref", "not finalising %q because thread count = %d", channelID, tc)
		lg.Debugf("channel %s: still processing %d ref count", channelID, tc)
		return nil
	}
	trace.Logf(ctx, "ref", "= 0, channel %s finalise", channelID)
	lg.Debugf("channel %s: closing channel file", channelID)
	r, err := cv.recorder(channelID)
	if err != nil {
		return err
	}
	if err := r.Close(); err != nil {
		return err
	}
	cv.mu.Lock()
	defer cv.mu.Unlock()
	delete(cv.cw, channelID)
	if cv.onFinalise != nil {
		return cv.onFinalise(ctx, channelID)
	}
	return nil
}

func (cv *Conversations) Close() error {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	for _, r := range cv.cw {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}
