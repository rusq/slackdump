package expproc

import (
	"context"
	"runtime/trace"
	"sync"

	"github.com/rusq/slackdump/v2/internal/chunk/processor"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/slack-go/slack"
)

// Conversations is a processor that writes the channel and thread messages.
type Conversations struct {
	dir string
	cw  map[string]*channelproc
	mu  sync.RWMutex
	lg  logger.Interface

	// fileSubproc is the files subprocessor, it is called by the Files method
	// in addition to recording the files in the chunk file (if recordFiles is
	// set).  It it useful, when one needs to download the files directly into
	// a final archive/directory, avoiding the intermediate step of
	// downloading files into the temporary directory, and then using
	// transform to download the files.
	fileSubproc processor.Filer // files sub-processor
	recordFiles bool

	onFinalise func(ctx context.Context, id string) error
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

// NewConversation returns the new conversation processor.  filesSubproc will
// be called for each file chunk.
func NewConversation(dir string, filesSubproc processor.Filer, opts ...ConvOption) (*Conversations, error) {
	c := &Conversations{
		dir:         dir,
		lg:          logger.Default,
		cw:          make(map[string]*channelproc),
		fileSubproc: filesSubproc,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// ensure ensures that the channel file is open and the recorder is
// initialized.
func (cv *Conversations) ensure(id string) error {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	if _, ok := cv.cw[id]; ok {
		return nil
	}
	bp, err := newBaseProc(cv.dir, id)
	if err != nil {
		return err
	}
	cv.cw[id] = &channelproc{
		baseproc: bp,
		refcnt:   1, // the channel itself is a reference
	}
	return nil
}

// ChannelInfo is called for each channel that is retrieved.
func (cv *Conversations) ChannelInfo(ctx context.Context, ci *slack.Channel, threadTS string) error {
	r, err := cv.recorder(ci.ID)
	if err != nil {
		return err
	}
	return r.ChannelInfo(ctx, ci, threadTS)
}

func (cv *Conversations) recorder(id string) (*baseproc, error) {
	cv.mu.RLock()
	r, ok := cv.cw[id]
	cv.mu.RUnlock()
	if ok {
		return r.baseproc, nil
	}
	if err := cv.ensure(id); err != nil {
		return nil, err
	}
	cv.mu.RLock()
	defer cv.mu.RUnlock()
	return cv.cw[id].baseproc, nil
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

	lg := logger.FromContext(ctx)
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
func (cv *Conversations) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	if err := cv.fileSubproc.Files(ctx, channel, parent, ff); err != nil {
		return err
	}
	if cv.recordFiles {
		r, err := cv.recorder(channel.ID)
		if err != nil {
			return err
		}
		if err := r.Files(ctx, channel, parent, ff); err != nil {
			return err
		}
	}
	return nil
}

func mkID(channelID, threadTS string, threadOnly bool) string {
	if !threadOnly {
		return channelID
	}
	return channelID + "-" + threadTS
}

// ThreadMessages is called for each of the thread messages that are
// retrieved. The parent message is passed in as well.
func (cv *Conversations) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly bool, isLast bool, tm []slack.Message) error {
	ctx, task := trace.NewTask(ctx, "ThreadMessages")
	defer task.End()
	lg := logger.FromContext(ctx)

	id := mkID(channelID, parent.ThreadTimestamp, threadOnly)
	r, err := cv.recorder(id)
	if err != nil {
		return err
	}
	if err := r.ThreadMessages(ctx, channelID, parent, threadOnly, isLast, tm); err != nil {
		return err
	}
	cv.decRef(id)
	refcnt := cv.refcount(id)
	trace.Logf(ctx, "ref", "decremented, current=%d", refcnt)
	lg.Debugf("processor: decreased ref count for %q to %d", id, refcnt)
	if isLast {
		trace.Log(ctx, "isLast", "true")
		lg.Debugf("processor: isLast=true, finalising thread %s", id)
		return cv.finalise(ctx, id)
	}
	return nil
}

// finalise closes the channel file if there are no more threads to process.
func (cv *Conversations) finalise(ctx context.Context, id string) error {
	lg := logger.FromContext(ctx)
	if tc := cv.refcount(id); tc > 0 {
		trace.Logf(ctx, "ref", "not finalising %q because ref count = %d", id, tc)
		lg.Debugf("channel %s: still processing %d ref count", id, tc)
		return nil
	}
	trace.Logf(ctx, "ref", "= 0, id %s finalise", id)
	lg.Debugf("id %s: closing channel file", id)
	r, err := cv.recorder(id)
	if err != nil {
		return err
	}
	if err := r.Close(); err != nil {
		return err
	}
	cv.mu.Lock()
	defer cv.mu.Unlock()
	delete(cv.cw, id)
	if cv.onFinalise != nil {
		return cv.onFinalise(ctx, id)
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
