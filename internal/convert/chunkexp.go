package convert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/transform"
	"github.com/rusq/slackdump/v2/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/slack-go/slack"
)

var (
	ErrEmptyChunk    = errors.New("missing chunk")
	ErrNoLocFunction = errors.New("missing location function")
)

// ChunkToExport is a converter between Chunk and Export formats.
type ChunkToExport struct {
	// src is the source directory with chunks
	src *chunk.Directory
	// trg is the target FS for the export
	trg fsadapter.FS
	// UploadDir is the upload directory name (relative to Src)
	includeFiles bool
	// FindFile should return the path to the file within the upload directory
	srcFileLoc func(*slack.Channel, *slack.File) string
	trgFileLoc func(*slack.Channel, *slack.File) string

	lg logger.Interface

	request chan filereq
	result  chan copyresult
}

type C2EOption func(*ChunkToExport)

// WithIncludeFiles sets the IncludeFiles option.
func WithIncludeFiles(b bool) C2EOption {
	return func(c *ChunkToExport) {
		c.includeFiles = b
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
func WithLogger(lg logger.Interface) C2EOption {
	return func(c *ChunkToExport) {
		if lg != nil {
			c.lg = lg
		}
	}
}

func NewChunkToExport(src *chunk.Directory, trg fsadapter.FS, opt ...C2EOption) *ChunkToExport {
	c := &ChunkToExport{
		src:          src,
		trg:          trg,
		includeFiles: false,
		srcFileLoc:   fileproc.MattermostFilepath,
		trgFileLoc:   fileproc.MattermostFilepath,
		lg:           logger.Default,
		request:      make(chan filereq, 1),
		result:       make(chan copyresult, 1),
	}
	for _, o := range opt {
		o(c)
	}
	return c
}

// Validate validates the input parameters.
func (c *ChunkToExport) Validate() error {
	if c.src == nil || c.trg == nil {
		return errors.New("convert: source and target must be set")
	}
	if c.includeFiles {
		const format = "convert: internal error: %s: %w"
		if c.srcFileLoc == nil {
			return fmt.Errorf(format, "source", ErrNoLocFunction)
		}
		if c.trgFileLoc == nil {
			return fmt.Errorf(format, "target", ErrNoLocFunction)
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

// Convert converts the chunk directory contents to the export format.
// It validates the input parameters.
//
// # Restrictions
//
// Currently, one chunk file per channel is supported.  If there are multiple
// chunk files per channel, the behaviour is undefined, but I expect it to
// overwrite the previous files.
func (c *ChunkToExport) Convert(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "convert.ChunkToExport")
	defer task.End()

	lg := logger.FromContext(ctx)

	if err := c.Validate(); err != nil {
		return err
	}
	channels, err := c.src.Channels()
	if err != nil {
		return err
	}
	users, err := c.src.Users()
	if err != nil {
		return err
	}
	var tfopts = []transform.ExpCvtOption{
		transform.ExpWithUsers(users),
	}
	if c.includeFiles {
		tfopts = append(tfopts, transform.ExpWithMsgUpdateFunc(func(ch *slack.Channel, m *slack.Message) error {
			// copy in a separate goroutine to avoid blocking the transform in
			// case of a synchronous fsadapter (e.g. zip file adapter can
			// write only one file at a time).
			c.request <- filereq{
				channel: ch,
				message: m,
			}
			return nil
		}))
		go c.copyworker(c.result, c.request)
	}

	conv := transform.NewExpConverter(c.src, c.trg, tfopts...)

	errC := make(chan error, 1)
	go func() {
		defer close(c.result)
		for _, ch := range channels {
			lg.Debugf("processing channel %q", ch.ID)
			if err := conv.Convert(ctx, chunk.ToFileID(ch.ID, "", false)); err != nil {
				errC <- fmt.Errorf("converter: failed to process %q: %w", ch.ID, err)
				return
			}
		}
	}()

LOOP:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errC:
			return err
		case res, more := <-c.result:
			if !more {
				break LOOP
			}
			if res.err != nil {
				return fmt.Errorf("error processing message with ts=%s: %w", res.fr.message.Timestamp, res.err)
			}
		}
	}

	lg.Debugf("writing index for %s", c.src.Name())
	if err := conv.WriteIndex(); err != nil {
		return err
	}
	return nil
}

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
		if !fileproc.IsValid(&f) {
			continue
		}
		srcpath := filepath.Join(c.src.Name(), c.srcFileLoc(ch, &f))
		if _, err := os.Stat(srcpath); err != nil {
			return fmt.Errorf("file ID=%s: %w", f.ID, err)
		}
		if _, err := os.Stat(srcpath); err != nil {
			return fmt.Errorf("file ID=%s: %w", f.ID, err)
		}
		trgpath := c.trgFileLoc(ch, &f)
		c.lg.Debugf("copying %q to %q", srcpath, trgpath)
		if err := c.copy2trg(trgpath, srcpath); err != nil {
			return fmt.Errorf("file ID=%s: %w", f.ID, err)
		}
	}
	return nil
}

// copy2trg copies the file from the source path to the target path.  Source
// path is absolute, target path is relative to the target FS adapter root.
func (c *ChunkToExport) copy2trg(trgpath, srcpath string) error {
	in, err := os.Open(srcpath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := c.trg.Create(trgpath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

type filereq struct {
	channel *slack.Channel
	message *slack.Message
}

type copyresult struct {
	fr  filereq
	err error
}

func (cr copyresult) Error() string {
	return fmt.Sprintf("copy: %s: %s", cr.fr.channel.Name, cr.err)
}

func (cr copyresult) Unwrap() error {
	return cr.err
}

func (c *ChunkToExport) copyworker(res chan<- copyresult, req <-chan filereq) {
	for fr := range req {
		res <- copyresult{
			fr:  fr,
			err: c.fileCopy(fr.channel, fr.message),
		}
	}
}
