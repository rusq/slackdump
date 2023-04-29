package expproc

import (
	"context"
	"errors"
	"runtime/trace"
	"sync"

	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/processor"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/slack-go/slack"
)

// Transformer is an interface that is called when the processor is finished
// processing a channel or thread.
type Transformer interface {
	// Transform is the function that starts the tranformation of the channel
	// or thread with the given id.  It is called  when the reference count
	// for the channel id becomes zero (meaning, that there are no more chunks
	// to process).  It should return [transform.ErrClosed] if the transformer
	// is closed.
	Transform(ctx context.Context, id chunk.FileID) error
}

// Conversations is a processor that writes the channel and thread messages.
// Zero value is unusable.  Use [NewConversation] to create a new instance.
type Conversations struct {
	dir string
	cw  map[chunk.FileID]*channelproc
	mu  sync.RWMutex
	lg  logger.Interface

	// filer is the filer subprocessor, it is called by the Files method
	// in addition to recording the filer in the chunk file (if recordFiles is
	// set).  It it useful, when one needs to download the filer directly into
	// a final archive/directory, avoiding the intermediate step of
	// downloading filer into the temporary directory, and then using
	// transform to download the filer.
	filer       processor.Filer // files sub-processor
	recordFiles bool

	// tf is the channel transformer that is called for each channel.
	tf Transformer
}

// ConvOption is a function that configures the Conversations processor.
type ConvOption func(*Conversations)

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
// be called for each file chunk, tf will be called for each completed channel
// or thread, when the reference count becomes zero.
// Reference count is increased with each call to Channel processing functions.
func NewConversation(dir string, filesSubproc processor.Filer, tf Transformer, opts ...ConvOption) (*Conversations, error) {
	// validation
	if filesSubproc == nil {
		return nil, errors.New("internal error: files subprocessor is nil")
	} else if tf == nil {
		return nil, errors.New("internal error: transformer is nil")
	}

	c := &Conversations{
		dir:   dir,
		lg:    logger.Default,
		cw:    make(map[chunk.FileID]*channelproc),
		filer: filesSubproc,
		tf:    tf,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// ensure ensures that the channel file is open and the recorder is
// initialized.
func (cv *Conversations) ensure(id chunk.FileID) error {
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
	r, err := cv.recorder(chunk.ToFileID(ci.ID, threadTS, threadTS != ""))
	if err != nil {
		return err
	}
	return r.ChannelInfo(ctx, ci, threadTS)
}

func (cv *Conversations) recorder(id chunk.FileID) (*baseproc, error) {
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
func (cv *Conversations) refcount(id chunk.FileID) int {
	cv.mu.RLock()
	defer cv.mu.RUnlock()
	if _, ok := cv.cw[id]; !ok {
		return 0
	}
	return cv.cw[id].refcnt
}

func (cv *Conversations) incRefN(id chunk.FileID, n int) {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	if _, ok := cv.cw[id]; !ok {
		return
	}
	cv.cw[id].refcnt += n
}

func (cv *Conversations) decRef(id chunk.FileID) {
	cv.mu.Lock()
	defer cv.mu.Unlock()
	if _, ok := cv.cw[id]; !ok {
		return
	}
	cv.cw[id].refcnt--
}

// Messages is called for each message that is retrieved.
func (cv *Conversations) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, mm []slack.Message) error {
	ctx, task := trace.NewTask(ctx, "Messages")
	defer task.End()

	lg := logger.FromContext(ctx)
	lg.Debugf("processor: channelID=%s, numThreads=%d, isLast=%t, len(mm)=%d", channelID, numThreads, isLast, len(mm))

	id := chunk.ToFileID(channelID, "", false)
	r, err := cv.recorder(id)
	if err != nil {
		return err
	}
	if numThreads > 0 {
		cv.incRefN(id, numThreads) // one for each thread
		trace.Logf(ctx, "ref", "added %d", numThreads)
		lg.Debugf("processor: increased ref count for %q to %d", channelID, cv.refcount(id))
	}
	if err := r.Messages(ctx, channelID, numThreads, isLast, mm); err != nil {
		return err
	}
	if isLast {
		trace.Log(ctx, "isLast", "true, decrease ref count")
		cv.decRef(id)
		return cv.finalise(ctx, id)
	}
	return nil
}

// Files is called for each file that is retrieved. The parent message is
// passed in as well.
func (cv *Conversations) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	if err := cv.filer.Files(ctx, channel, parent, ff); err != nil {
		return err
	}
	if cv.recordFiles {
		id := chunk.ToFileID(channel.ID, parent.ThreadTimestamp, false) // we don't do files for threads in export
		r, err := cv.recorder(id)
		if err != nil {
			return err
		}
		if err := r.Files(ctx, channel, parent, ff); err != nil {
			return err
		}
	}
	return nil
}

// ThreadMessages is called for each of the thread messages that are
// retrieved. The parent message is passed in as well.
func (cv *Conversations) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly bool, isLast bool, tm []slack.Message) error {
	ctx, task := trace.NewTask(ctx, "ThreadMessages")
	defer task.End()
	lg := logger.FromContext(ctx)

	id := chunk.ToFileID(channelID, parent.ThreadTimestamp, threadOnly)
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
func (cv *Conversations) finalise(ctx context.Context, id chunk.FileID) error {
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
	if cv.tf != nil {
		return cv.tf.Transform(ctx, id)
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
