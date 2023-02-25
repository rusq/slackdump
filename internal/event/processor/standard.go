package processor

import (
	"context"
	"io"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/internal/event"
	"github.com/slack-go/slack"
)

type Standard struct {
	*event.Recorder
	dl *downloader.Client

	opts options
}

// NewStandard creates a new standard processor.  It will write the output to
// the given writer.  The downloader is used to download files.  The directory
// is the directory where the files will be downloaded to.  The options are
// functional options.  See the NoFiles option.
func NewStandard(ctx context.Context, w io.Writer, sess downloader.Downloader, dir string, opts ...Option) (*Standard, error) {
	opt := options{dumpFiles: false}
	for _, o := range opts {
		o(&opt)
	}

	dl := downloader.New(sess, fsadapter.NewDirectory(dir))
	dl.Start(ctx)

	r := event.NewRecorder(w)
	return &Standard{
		Recorder: r,
		dl:       dl,
		opts:     opt,
	}, nil
}

// Files implements the Processor interface. It will download files if the
// dumpFiles option is enabled.
func (s *Standard) Files(ctx context.Context, channelID string, parent slack.Message, isThread bool, ff []slack.File) error {
	if !s.opts.dumpFiles {
		// ignore files if requested
		return nil
	}
	// custom file processor, because we need to donwload those files
	for i := range ff {
		filename, err := s.dl.DownloadFile(channelID, ff[i])
		if err != nil {
			return err
		}
		s, _ := s.Recorder.State()
		s.AddFile(channelID, ff[i].ID, filename)
	}
	return nil
}

func (s *Standard) Close() error {
	s.dl.Stop()
	return s.Recorder.Close()
}
