package processor

import (
	"context"
	"io"
	"runtime/trace"

	"github.com/rusq/fsadapter"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/internal/chunk"
)

type Standard struct {
	*chunk.Recorder
	dl *downloader.ClientV1

	opts options
}

type options struct {
	dumpFiles bool
}

// Option is a functional option for the processor.
type Option func(*options)

// DumpFiles disables the file processing (enabled by default).  It may be
// useful on enterprise workspaces where the file download may be monitored.
// See [#191]
//
// [#191]: https://github.com/rusq/slackdump/discussions/191#discussioncomment-4953235
func DumpFiles(b bool) Option {
	return func(o *options) {
		o.dumpFiles = b
	}
}

// NewStandard creates a new standard processor.  It will write the output to
// the given writer.  The downloader is used to download files.  The directory
// is the directory where the files will be downloaded to.  The options are
// functional options.  See the NoFiles option.
func NewStandard(ctx context.Context, w io.Writer, sess downloader.Downloader, dir string, opts ...Option) (*Standard, error) {
	opt := options{dumpFiles: true}
	for _, o := range opts {
		o(&opt)
	}

	dl := downloader.NewV1(sess, fsadapter.NewDirectory(dir))
	dl.Start(ctx)

	r := chunk.NewRecorder(w)
	return &Standard{
		Recorder: r,
		dl:       dl,
		opts:     opt,
	}, nil
}

// Files implements the Processor interface. It will download files if the
// dumpFiles option is enabled.
func (s *Standard) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, isThread bool, ff []slack.File) error {
	if !s.opts.dumpFiles {
		// ignore files if requested
		return nil
	}
	st, err := s.State()
	if err != nil {
		return err
	}
	// custom file processor, because we need to donwload those files
	for i := range ff {
		if mode := ff[i].Mode; mode == "hidden_by_limit" || mode == "external" || ff[i].IsExternal {
			// ignore files that are hidden by the limit
			trace.Logf(ctx, "skip", "unfetchable file type: %q", ff[i].ID)
			continue
		}
		filename, err := s.dl.DownloadFile(channel.ID, ff[i])
		if err != nil {
			return err
		}
		st.AddFile(channel.ID, ff[i].ID, filename)
	}
	return nil
}

func (s *Standard) Close() error {
	s.dl.Stop()
	return s.Recorder.Close()
}
