package dirproc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"runtime/trace"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/processor"
)

//go:generate mockgen -source=conversations.go -destination=dirproc_mock_test.go -package=dirproc

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
	t  tracker
	lg *slog.Logger

	// subproc is the files subprocessor, it is called by the Files method
	// in addition to recording the files in the chunk file (if recordFiles is
	// set).  It it useful, when one needs to download the file directly into
	// a final archive/directory, avoiding the intermediate step of
	// downloading files into the temporary directory, and then using
	// transform to download the files.
	subproc     processor.Filer // files sub-processor
	recordFiles bool

	// tf is the channel transformer that is called for each channel.
	tf Transformer
}

// tracker is an interface for a recorder of data.

type tracker interface {
	Recorder(id chunk.FileID) (datahandler, error)
	RefCount(id chunk.FileID) int
	Unregister(id chunk.FileID) error
	CloseAll() error
}

// datahandler is an interface for the data processor
type datahandler interface {
	processor.ChannelInformer
	processor.Messenger
	processor.Filer
	counter
	io.Closer
}

type counter interface {
	Inc() int
	Dec() int
	Add(int) int
	N() int
}

// ConvOption is a function that configures the Conversations processor.
type ConvOption func(*Conversations)

// WithLogger sets the logger for the processor.
func WithLogger(lg *slog.Logger) ConvOption {
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

var (
	errNilSubproc     = errors.New("internal error: files subprocessor is nil")
	errNilTransformer = errors.New("internal error: transformer is nil")
)

// NewConversation returns the new conversation processor.  filesSubproc will
// be called for each file chunk, tf will be called for each completed channel
// or thread, when the reference count becomes zero.
// Reference count is increased with each call to Channel processing functions.
func NewConversation(cd *chunk.Directory, filesSubproc processor.Filer, tf Transformer, opts ...ConvOption) (*Conversations, error) {
	// validation
	if filesSubproc == nil {
		return nil, errNilSubproc
	} else if tf == nil {
		return nil, errNilTransformer
	}

	c := &Conversations{
		t:       newFileTracker(cd),
		lg:      slog.Default(),
		subproc: filesSubproc,
		tf:      tf,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// ChannelInfo is called for each channel that is retrieved.
func (cv *Conversations) ChannelInfo(ctx context.Context, ci *slack.Channel, threadTS string) error {
	r, err := cv.t.Recorder(chunk.ToFileID(ci.ID, threadTS, threadTS != ""))
	if err != nil {
		return err
	}
	return r.ChannelInfo(ctx, ci, threadTS)
}

// Messages is called for each message slice that is retrieved.
func (cv *Conversations) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, mm []slack.Message) error {
	ctx, task := trace.NewTask(ctx, "Messages")
	defer task.End()

	lg := cv.lg.With("in", "Messages", "channel_id", channelID, "num_threads", numThreads, "is_last", isLast, "len_messages", len(mm))
	lg.Debug("started")
	cv.debugtrace(ctx, "%s: Messages: numThreads=%d, isLast=%t, len(mm)=%d", channelID, numThreads, isLast, len(mm))

	id := chunk.ToFileID(channelID, "", false)
	r, err := cv.t.Recorder(id)
	if err != nil {
		return err
	}
	n := r.Add(numThreads)

	cv.debugtrace(ctx, "%s: Messages: increased by %d to %d", channelID, numThreads, n)
	lg.DebugContext(ctx, "count increased", "by", numThreads, "current", n)

	if err := r.Messages(ctx, channelID, numThreads, isLast, mm); err != nil {
		return err
	}

	if isLast {
		n := r.Dec()
		cv.debugtrace(ctx, "%s: Messages: decreased by 1 to %d, finalising", channelID, n)
		lg.DebugContext(ctx, "count decreased", "by", 1, "current", n)
		return cv.finalise(ctx, id)
	}
	return nil
}

// ThreadMessages is called for each of the thread messages that are
// retrieved. The parent message is passed in as well.
func (cv *Conversations) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly bool, isLast bool, tm []slack.Message) error {
	ctx, task := trace.NewTask(ctx, "ThreadMessages")
	defer task.End()

	lg := cv.lg.With("in", "ThreadMessages", "channel_id", channelID, "parent_ts", parent.ThreadTimestamp, "is_last", isLast, "len(tm)", len(tm))
	lg.Debug("started")
	cv.debugtrace(ctx, "%s: ThreadMessages: parent=%s, isLast=%t, len(tm)=%d", channelID, parent.ThreadTimestamp, isLast, len(tm))

	id := chunk.ToFileID(channelID, parent.ThreadTimestamp, threadOnly)
	r, err := cv.t.Recorder(id)
	if err != nil {
		return err
	}
	if err := r.ThreadMessages(ctx, channelID, parent, threadOnly, isLast, tm); err != nil {
		return err
	}
	if isLast {
		n := r.Dec()
		lg.DebugContext(ctx, "count decreased, finalising", "by", 1, "current", n)
		cv.debugtrace(ctx, "%s:%s: ThreadMessages: decreased by 1 to %d, finalising", id, parent.Timestamp, n)
		return cv.finalise(ctx, id)
	}
	return nil
}

// finalise closes the channel file if there are no more threads to process.
func (cv *Conversations) finalise(ctx context.Context, id chunk.FileID) error {
	lg := cv.lg.With("in", "finalise", "file_id", id)
	if tc := cv.t.RefCount(id); tc > 0 {
		lg.DebugContext(ctx, "not finalising", "ref_count", tc)
		cv.debugtrace(ctx, "%s: finalise: not finalising, ref count = %d", id, tc)
		return nil
	}
	lg.Debug("finalising", "ref_count", 0)
	cv.debugtrace(ctx, "%s: finalise: ref count = 0, finalising...", id)
	if err := cv.t.Unregister(id); err != nil {
		return err
	}
	if cv.tf != nil {
		return cv.tf.Transform(ctx, id)
	}
	return nil
}

// Files is called for each file that is retrieved. The parent message is
// passed in as well.
func (cv *Conversations) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	if err := cv.subproc.Files(ctx, channel, parent, ff); err != nil {
		return err
	}
	if !cv.recordFiles {
		return nil
	}
	id := chunk.ToFileID(channel.ID, parent.ThreadTimestamp, false) // we don't do files for threads in export
	r, err := cv.t.Recorder(id)
	if err != nil {
		return err
	}
	if err := r.Files(ctx, channel, parent, ff); err != nil {
		return err
	}
	return nil
}

func (cv *Conversations) ChannelUsers(ctx context.Context, channelID string, threadTS string, cu []string) error {
	r, err := cv.t.Recorder(chunk.ToFileID(channelID, threadTS, threadTS != ""))
	if err != nil {
		return err
	}
	return r.ChannelUsers(ctx, channelID, threadTS, cu)
}

func (cv *Conversations) Close() error {
	return cv.t.CloseAll()
}

func (cv *Conversations) debugtrace(ctx context.Context, fmt string, args ...any) {
	trace.Logf(ctx, "debug", fmt, args...)
}
