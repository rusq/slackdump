package convert

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/trace"
	"sync"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/convert/transform"
	"github.com/rusq/slackdump/v3/internal/convert/transform/fileproc"
	"github.com/rusq/slackdump/v3/source"
)

const (
	defWorkers = 8 // default number of goroutines to process channels
)

var (
	ErrEmptyChunk    = errors.New("missing chunk")
	ErrNoLocFunction = errors.New("missing location function")
)

// ToExport is a converter between Chunk and Export formats.  Zero value
// is not usable.
type ToExport struct {
	// src is the source
	src source.Sourcer
	// trg is the target FS for the export
	trg  fsadapter.FS
	opts options

	workers int // number of workers to use to convert channels

	filerequest chan copyrequest
	fileresult  chan copyresult
	avtrresult  chan copyresult
}

// NewToExport returns the converter from any source to export format. src is the Source
// to be converted (e.g. chunk or database), trgfs is the target FS adapter where files
// and data will be written.  By default, the converter does not include files and
// avatars.  The default storage format for both source and destination is Mattermost, use
// functional options to configure this behaviour.
func NewToExport(src source.Sourcer, trg fsadapter.FS, opt ...Option) *ToExport {
	c := &ToExport{
		src: src,
		trg: trg,
		opts: options{
			includeFiles:   false,
			includeAvatars: false,
			trgFileLoc:     source.MattermostFilepath,
			avtrFileLoc:    fileproc.AvatarPath,
			lg:             slog.Default(),
		},
		filerequest: make(chan copyrequest, 1),
		fileresult:  make(chan copyresult, 1),
		avtrresult:  make(chan copyresult, 1),
		workers:     defWorkers,
	}
	for _, o := range opt {
		o(&c.opts)
	}
	return c
}

// Validate validates the input parameters.
func (c *ToExport) Validate() error {
	if c.src == nil || c.trg == nil {
		return errors.New("convert: source and target must be set")
	}
	if err := c.opts.Validate(); err != nil {
		return fmt.Errorf("convert: %w", err)
	}
	// users chunk is required
	if _, err := c.src.Users(context.Background()); err != nil {
		return fmt.Errorf("convert: error getting users: %w", err)
	}

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
func (c *ToExport) Convert(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "convert.ChunkToExport")
	defer task.End()

	if err := c.Validate(); err != nil {
		return err
	}
	channels, err := c.src.Channels(ctx)
	if err != nil {
		return err
	}
	users, err := c.src.Users(ctx)
	if err != nil {
		return err
	}

	tfopts := []transform.ExpCvtOption{
		transform.ExpWithUsers(users),
	}
	// 1. generator
	chC := sliceToChan(channels)

	lg := c.opts.lg

	errC := make(chan error, c.workers)
	{
		// 2. workers
		var filewg sync.WaitGroup

		if c.opts.includeFiles && c.src.Files().Type() != source.STnone {
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

		if c.opts.includeAvatars && c.src.Avatars().Type() != source.STnone {
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
					lg := lg.With("channel", ch.ID)
					lg.Debug("processing channel")
					if err := conv.Convert(ctx, ch.ID, ""); err != nil {
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
			lg.DebugContext(ctx, "writing index", "name", c.src.Name())
			if err := conv.WriteIndex(ctx); err != nil {
				errC <- err
			}
		}()
		// 2.3. workers sentinels
		go func() {
			msgwg.Wait()
			lg.Debug("messages wait group done, closing file requests")
			close(c.filerequest)
			filewg.Wait()
			lg.Debug("file workers done, finalising")
			close(errC)
		}()
	}
	// 3. result processor
	fileresults := merge(c.fileresult, c.avtrresult)
	go func() {
		for res := range fileresults {
			if res.err != nil {
				if res.fr.message != nil {
					lg.Error("file converter: error processing message", "ts", res.fr.message.Timestamp, "err", res.err)
				} else {
					lg.Error("file converter", "err", res.err)
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
				lg.ErrorContext(ctx, "worker", "error", err)
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

func (c *ToExport) copyworker(req <-chan copyrequest) {
	defer close(c.fileresult)
	fc := FileCopier{
		src:      c.src,
		trg:      c.trg,
		trgLocFn: c.opts.trgFileLoc,
		enabled:  c.opts.includeFiles,
	}
	c.opts.lg.Debug("copy worker started")
	for r := range req {
		c.fileresult <- copyresult{
			fr:  r,
			err: fc.Copy(r.channel, r.message),
		}
	}
	c.opts.lg.Debug("copy worker done")
}

func (c *ToExport) avatarWorker(users []slack.User) {
	lg := c.opts.lg
	lg.Debug("avatar worker started")
	defer close(c.avtrresult)
	for _, u := range users {
		if u.Profile.ImageOriginal == "" {
			continue
		}
		lg.Debug("processing avatar", "user", u.ID)
		loc := c.opts.avtrFileLoc(&u)
		fsys := c.src.Avatars().FS()
		srcLoc, err := c.src.Avatars().File(source.AvatarParams(&u))
		if err != nil {
			err = fmt.Errorf("error getting avatar for user %s: %w", u.ID, err)
		} else {
			err = copy2trg(c.trg, loc, fsys, srcLoc)
			if err != nil {
				err = fmt.Errorf("error copying avatar for user %s: %w", u.ID, err)
			}
		}
		c.avtrresult <- copyresult{
			err: err,
		}
		if err != nil {
			continue
		}
		lg.Debug("avatar processed", "user", u.ID)
	}
	lg.Debug("avatar worker done")
}
