package transform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/rusq/slack"
)

type UserConverter interface {
	Converter
	SetUsers([]slack.User)
	HasUsers() bool
}

// ExportCoordinator is a takes the chunks produced by the
// processor and transforms them into a Slack Export format.  It is suitable
// for async processing, in which case, OnFinalise function is passed to the
// processor, the finalisation requests will be queued (up to a
// [bufferSz]) and will be processed once Start or StartWithUsers is called.
//
// Please note, that transform requires users to be passed either through
// options or through StartWithUsers.  If users are not passed, the
// [ExportCoordinator.Start] will return an error.
//
// The asynchronous pattern to run the transform is as follows:
//
//  1. Create the transform instance.
//  2. Defer its Close method.
//  3. In goroutine: Start user processing, and in the same goroutine, after
//     all users are fetched, call [ExportCoordinator.StartWithUsers], passing
//     the fetched users slice.
//  4. In another goroutine, start the ExportCoordinator Conversation
//     processor, passing the transformer's OnFinalise function as the
//     Finaliser option.  It will be called by export processor for each
//     channel that was completed.
type ExportCoordinator struct {
	cvt    UserConverter
	lg     *slog.Logger
	closed atomic.Bool

	start    chan struct{}
	err      chan error   // error channel used to propagate errors to the main thread.
	requestC chan request // channel used to pass channel IDs to the worker.
}

// bufferSz is the default size of the channel IDs buffer.  This is the number
// of channel IDs that will be queued without blocking before the
// [transform.Export] is started.
const bufferSz = 100

// ExpOption is a function that configures the Export instance.
type ExpOption func(*ExportCoordinator)

// WithBufferSize sets the size of the channel IDs buffer.  This is the number
// of channel IDs that will be queued without blocking before the [transform.Export] is
// started.
func WithBufferSize(n int) ExpOption {
	return func(t *ExportCoordinator) {
		if n < 1 {
			n = bufferSz
		}
		t.requestC = make(chan request, n)
	}
}

// WithUsers allows to pass a list of users to the transform.
func WithUsers(users []slack.User) ExpOption {
	return func(t *ExportCoordinator) {
		t.cvt.SetUsers(users)
	}
}

// NewExportCoordinator creates a new ExportCoordinator instance.
func NewExportCoordinator(ctx context.Context, cvt UserConverter, tfopt ...ExpOption) *ExportCoordinator {
	t := &ExportCoordinator{
		cvt:      cvt,
		lg:       slog.Default(),
		start:    make(chan struct{}),
		requestC: make(chan request, bufferSz),
		err:      make(chan error, 1),
	}
	for _, opt := range tfopt {
		opt(t)
	}

	// will hold till something is sent into start channel (usually by Start method
	go t.worker(ctx)

	return t
}

// StartWithUsers starts the Transform processor with the provided list of
// users.  Users are used to populate each message with the user profile, as
// per Slack original export format.
func (t *ExportCoordinator) StartWithUsers(ctx context.Context, users []slack.User) error {
	if len(users) == 0 {
		return errors.New("users list is empty or nil")
	}
	t.cvt.SetUsers(users)
	return t.Start(ctx)
}

// Start starts the coordinator, the users must have been initialised with the
// WithUsers option.  Otherwise, use [ExportCoordinator.StartWithUsers] method.
// The function doesn't check if coordinator was already started or not.
func (t *ExportCoordinator) Start(ctx context.Context) error {
	t.lg.DebugContext(ctx, "transform: starting transform")
	if !t.cvt.HasUsers() {
		return errors.New("internal error: users not initialised")
	}
	if t.closed.Load() {
		return errors.New("transform is closed")
	}
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case err := <-t.err:
		return fmt.Errorf("transform: pending error: %w", err)
	default:
		t.start <- struct{}{}
	}

	return nil
}

// Transform is the function that should be passed to the Channel processor.
// It will not block if the internal buffer is full.  Buffer size can be
// set with the WithBufferSize option.  The caller is allowed to call OnFinalise
// even if the processor is not started, in which case the channel ID will
// be queued for processing once the processor is started.  If the export
// worker is closed, it will return ErrClosed.
func (t *ExportCoordinator) Transform(ctx context.Context, channelID, threadTS string) error {
	select {
	case err := <-t.err:
		return err
	default:
	}
	if t.closed.Load() {
		return ErrClosed
	}
	t.lg.Debug("transform: placing request in the queue", "channel_id", channelID, "thread_ts", threadTS)
	t.requestC <- request{channelID, threadTS}
	return nil
}

func (t *ExportCoordinator) worker(ctx context.Context) {
	defer close(t.err)

	lg := t.lg.With("in", "ExportCoordinator.worker")

	lg.Debug("worker waiting", "buffer_size", cap(t.requestC))
	<-t.start
	lg.Debug("worker started", "queue_size", len(t.requestC))
	for req := range t.requestC {
		lg.Debug("transforming channel", "channel_id", req)
		if err := t.cvt.Convert(ctx, req.channelID, req.threadTS); err != nil {
			lg.Debug("transforming channel failure", "channel_id", req, "error", err)
			t.err <- err
			continue
		}
	}
}

// Close closes the coordinator.  It must be called once it is guaranteed that
// [Transform] will not be called anymore, otherwise the call to Transform
// will panic with "send on the closed channel". If the coordinator is already
// closed, it will return nil.
func (t *ExportCoordinator) Close() (err error) {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}
	t.lg.Debug("transform: closing transform")
	close(t.requestC)
	close(t.start)
	t.lg.Debug("transform: waiting for workers to finish")

	return <-t.err
}
