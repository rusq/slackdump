package convert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/trace"
	"sync"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
)

const (
	defWorkers = 8 // default number of goroutines to process channels
)

var (
	ErrEmptyChunk    = errors.New("missing chunk")
	ErrNoLocFunction = errors.New("missing location function")
)

// ChunkToExport is a converter between Chunk and Export formats.  Zero value
// is not usable.
type ChunkToExport struct {
	// src is the source directory with chunks
	src *chunk.Directory
	// trg is the target FS for the export
	trg fsadapter.FS
	// includeFiles is a flag to include files in the export
	includeFiles bool
	// includeAvatars is a flag to include avatars in the export
	includeAvatars bool
	// srcFileLoc should return the file location within the source directory.
	srcFileLoc func(*slack.Channel, *slack.File) string
	// trgFileLoc should return the file location within the target directory
	trgFileLoc func(*slack.Channel, *slack.File) string
	// avtrFileLoc should return the avatar file location.
	avtrFileLoc func(*slack.User) string

	lg *slog.Logger

	workers int // number of workers to use to convert channels

	filerequest chan copyrequest
	fileresult  chan copyresult
	avtrresult  chan copyresult
}

type C2EOption func(*ChunkToExport)

// WithIncludeFiles sets the IncludeFiles option.
func WithIncludeFiles(b bool) C2EOption {
	return func(c *ChunkToExport) {
		c.includeFiles = b
	}
}

// WithIncludeAvatars sets the IncludeAvataars option.
func WithIncludeAvatars(b bool) C2EOption {
	return func(c *ChunkToExport) {
		c.includeAvatars = b
	}
}

// WithSrcFileLoc sets the SrcFileLoc function.
func WithSrcFileLoc(fn func(*slack.Channel, *slack.File) string) C2EOption {
	return func(c *ChunkToExport) {
		if fn != nil {
			c.srcFileLoc = fn
		}
	}
}

// WithTrgFileLoc sets the TrgFileLoc function.
func WithTrgFileLoc(fn func(*slack.Channel, *slack.File) string) C2EOption {
	return func(c *ChunkToExport) {
		if fn != nil {
			c.trgFileLoc = fn
		}
	}
}

// WithLogger sets the logger.
func WithLogger(lg *slog.Logger) C2EOption {
	return func(c *ChunkToExport) {
		if lg != nil {
			c.lg = lg
		}
	}
}

func NewChunkToExport(src *chunk.Directory, trg fsadapter.FS, opt ...C2EOption) *ChunkToExport {
	c := &ChunkToExport{
		src:            src,
		trg:            trg,
		includeFiles:   false,
		includeAvatars: false,
		srcFileLoc:     fileproc.MattermostFilepath,
		trgFileLoc:     fileproc.MattermostFilepath,
		avtrFileLoc:    fileproc.AvatarPath,
		lg:             slog.Default(),
		filerequest:    make(chan copyrequest, 1),
		fileresult:     make(chan copyresult, 1),
		avtrresult:     make(chan copyresult, 1),
		workers:        defWorkers,
	}
	for _, o := range opt {
		o(c)
	}
	return c
}

// Validate validates the input parameters.
func (c *ChunkToExport) Validate() error {
	const format = "convert: internal error: %s: %w"
	if c.src == nil || c.trg == nil {
		return errors.New("convert: source and target must be set")
	}
	if c.includeFiles {
		if c.srcFileLoc == nil {
			return fmt.Errorf(format, "source", ErrNoLocFunction)
		}
		if c.trgFileLoc == nil {
			return fmt.Errorf(format, "target", ErrNoLocFunction)
		}
	}
	if c.includeAvatars {
		if c.avtrFileLoc == nil {
			return fmt.Errorf(format, "avatar", ErrNoLocFunction)
		}
	}
	// users chunk is required
	if fi, err := c.src.Stat(chunk.FUsers); err != nil {
		return fmt.Errorf("users chunk: %w", err)
	} else if fi.Size() == 0 {
		return fmt.Errorf("users chunk: %w", ErrEmptyChunk)
	}
	// we are not checking for channels chunk because channel information will
	// be stored in each of the chunk files, and we can collect it from there.

	return nil
}

func sliceToChan[T any](s []T) <-chan T {
	ch := make(chan T)
	go func() {
		defer close(ch)
		for _, v := range s {
			ch <- v
		}
	}()
	return ch
}

// Convert converts the chunk directory contents to the export format. It
// validates the input parameters.
//
// # Restrictions
//
// TODO: Currently, one chunk file per channel is supported.  If there are
// multiple chunk files per channel, the behaviour is undefined, but I expect
// it to overwrite the previous files.
func (c *ChunkToExport) Convert(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "convert.ChunkToExport")
	defer task.End()

	if err := c.Validate(); err != nil {
		return err
	}
	channels, err := c.src.Channels(ctx)
	if err != nil {
		return err
	}
	users, err := c.src.Users()
	if err != nil {
		return err
	}

	tfopts := []transform.ExpCvtOption{
		transform.ExpWithUsers(users),
	}
	// 1. generator
	chC := sliceToChan(channels)

	errC := make(chan error, c.workers)
	{
		// 2. workers
		var filewg sync.WaitGroup

		if c.includeFiles {
			tfopts = append(tfopts, transform.ExpWithMsgUpdateFunc(func(ch *slack.Channel, m *slack.Message) error {
				// copy in a separate goroutine to avoid blocking the transform in
				// case of a synchronous fsadapter (e.g. zip file adapter can write
				// only one file at a time).
				c.filerequest <- copyrequest{
					channel: ch,
					message: m,
				}
				return nil
			}))
			filewg.Add(1)
			go func() {
				c.copyworker(c.filerequest)
				filewg.Done()
			}()
		} else {
			close(c.fileresult)
		}

		if c.includeAvatars {
			filewg.Add(1)
			go func() {
				c.avatarWorker(users)
				filewg.Done()
			}()
		} else {
			close(c.avtrresult)
		}

		// 2.1 converter
		var msgwg sync.WaitGroup
		conv := transform.NewExpConverter(c.src, c.trg, tfopts...)
		for range c.workers {
			msgwg.Add(1)
			go func() {
				defer msgwg.Done()
				for ch := range chC {
					lg := c.lg.With("channel", ch.ID)
					lg.Debug("processing channel")
					if err := conv.Convert(ctx, chunk.ToFileID(ch.ID, "", false)); err != nil {
						errC <- fmt.Errorf("converter: failed to process %q: %w", ch.ID, err)
						return
					}
				}
			}()
		}
		// 2.2 index writer
		msgwg.Add(1)
		go func() {
			defer msgwg.Done()
			c.lg.DebugContext(ctx, "writing index", "name", c.src.Name())
			if err := conv.WriteIndex(ctx); err != nil {
				errC <- err
			}
		}()
		// 2.3. workers sentinels
		go func() {
			msgwg.Wait()
			c.lg.Debug("messages wait group done, closing file requests")
			close(c.filerequest)
			filewg.Wait()
			c.lg.Debug("file workers done, finalising")
			close(errC)
		}()
	}
	// 3. result processor
	fileresults := merge(c.fileresult, c.avtrresult)
	go func() {
		for res := range fileresults {
			if res.err != nil {
				if res.fr.message != nil {
					c.lg.Error("file converter: error processing message", "ts", res.fr.message.Timestamp, "err", res.err)
				} else {
					c.lg.Error("file converter", "err", res.err)
				}
				errC <- res.err
			}
		}
	}()

	var failed bool
LOOP:
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case err, more := <-errC:
			if !more {
				break LOOP
			}
			if err != nil {
				slog.Error("worker error", "err", err)
				failed = true
			}
		}
	}
	if failed {
		return errors.New("convert: there were errors")
	}
	return nil
}

func merge(resC ...<-chan copyresult) <-chan copyresult {
	var wg sync.WaitGroup
	out := make(chan copyresult, 1)

	output := func(c <-chan copyresult) {
		for res := range c {
			out <- res
		}
		wg.Done()
	}
	wg.Add(len(resC))
	for _, c := range resC {
		go output(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

type copyerror struct {
	FileID string
	Err    error
}

func (e *copyerror) Error() string {
	return fmt.Sprintf("copy error: file ID=%s: %v", e.FileID, e.Err)
}

func (e *copyerror) Unwrap() error {
	return e.Err
}

// fileCopy iterates through the files in the message and copies them to the
// target directory.  Source file location is determined by calling the
// srcFileLoc function, joined with the chunk directory name.  target file
// location — by calling trgFileLoc function, and is relative to the target
// fsadapter root.
func (c *ChunkToExport) fileCopy(ch *slack.Channel, msg *slack.Message) error {
	if !c.includeFiles {
		return nil
	}
	if msg == nil {
		return errors.New("convert: internal error: callback: nil message")
	}
	if len(msg.Files) == 0 {
		// no files to process
		return nil
	}
	for _, f := range msg.Files {
		if err := fileproc.IsValidWithReason(&f); err != nil {
			c.lg.Info("skipping", "file", f.ID, "error", err)
			continue
		}

		srcpath := filepath.Join(c.src.Name(), c.srcFileLoc(ch, &f))
		trgpath := c.trgFileLoc(ch, &f)

		sfi, err := os.Stat(srcpath)
		if err != nil {
			return &copyerror{f.ID, err}
		}
		if sfi.Size() == 0 {
			c.lg.Warn("skipping", "file", f.ID, "reason", "empty file")
			continue
		}
		c.lg.Debug("copying", "srcpath", srcpath, "trgpath", trgpath)
		if err := copy2trg(c.trg, trgpath, srcpath); err != nil {
			return &copyerror{f.ID, err}
		}
	}
	return nil
}

// copy2trg copies the file from the source path to the target path.  Source
// path is absolute, target path is relative to the target FS adapter root.
func copy2trg(trgfs fsadapter.FS, trgpath, srcpath string) error {
	in, err := os.Open(srcpath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := trgfs.Create(trgpath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

type copyrequest struct {
	channel *slack.Channel
	message *slack.Message
}

type copyresult struct {
	fr  copyrequest
	err error
}

func (cr copyresult) Error() string {
	return fmt.Sprintf("copy error: channel [%s]: %s", cr.fr.channel.ID, cr.err)
}

func (cr copyresult) Unwrap() error {
	return cr.err
}

func (c *ChunkToExport) copyworker(req <-chan copyrequest) {
	defer close(c.fileresult)
	c.lg.Debug("copy worker started")
	for r := range req {
		c.fileresult <- copyresult{
			fr:  r,
			err: c.fileCopy(r.channel, r.message),
		}
	}
	c.lg.Debug("copy worker done")
}

func (c *ChunkToExport) avatarWorker(users []slack.User) {
	c.lg.Debug("avatar worker started")
	defer close(c.avtrresult)
	for _, u := range users {
		if u.Profile.ImageOriginal == "" {
			continue
		}
		c.lg.Debug("processing avatar", "user", u.ID)
		loc := c.avtrFileLoc(&u)
		err := copy2trg(c.trg, loc, filepath.Join(c.src.Name(), loc))
		if err != nil {
			err = fmt.Errorf("error copying avatar for user %s: %w", u.ID, err)
		}
		c.avtrresult <- copyresult{
			err: err,
		}
		if err != nil {
			continue
		}
		c.lg.Debug("avatar processed", "user", u.ID)
	}
	c.lg.Debug("avatar worker done")
}
