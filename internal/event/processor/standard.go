package processor

import (
	"os"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/internal/event"
	"github.com/slack-go/slack"
)

type Standard struct {
	*event.Recorder
	channelID string
	fs        fsadapter.FS
}

func NewStandard(channelID string, fs fsadapter.FS) (*Standard, error) {
	f, err := os.CreateTemp("", "slackdump-"+channelID+"-*.jsonl")
	if err != nil {
		return nil, err
	}
	r := event.NewRecorder(f)
	return &Standard{
		Recorder:  r,
		channelID: channelID,
		fs:        fs,
	}, nil
}

func (s *Standard) Files(par *slack.Message, f []slack.File) error {
	// custom file processor, because we need to donwload those files
	panic("implement me")
}

func (s *Standard) Close() error {
	// reconstruct the final json file
	panic("implement me")
}
