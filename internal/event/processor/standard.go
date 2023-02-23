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
}

func NewStandard(w io.Writer, sess downloader.Downloader, dir string) (*Standard, error) {
	r := event.NewRecorder(w)
	dl := downloader.New(sess, fsadapter.NewDirectory(dir))
	dl.Start(context.Background())
	return &Standard{
		Recorder: r,
		dl:       dl,
	}, nil
}

func (s *Standard) Files(channelID string, parent slack.Message, isThread bool, m []slack.File) error {
	// custom file processor, because we need to donwload those files
	for i := range m {
		if _, err := s.dl.DownloadFile(channelID, m[i]); err != nil {
			return err
		}
	}
	return nil
}

func fileUrls(ff []slack.File) []string {
	var urls = make([]string, 0, len(ff))
	for i := range ff {
		urls = append(urls, ff[i].URLPrivate)
	}
	return urls
}

func (s *Standard) Close() error {
	// reconstruct the final json file
	s.dl.Stop()
	return nil
}
