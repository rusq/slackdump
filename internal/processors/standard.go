package processors

import (
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/slack-go/slack"
)

type Standard struct {
	fs fsadapter.FS
}

func (s *Standard) Messages(m []slack.Message) error {
	panic("implement me")
}

func (s *Standard) Files(par *slack.Message, f []slack.File) error {
	panic("implement me")
}

func (s *Standard) ThreadMessages(par *slack.Message, tm []slack.Message) error {
	panic("implement me")
}

func (s *Standard) Close() error {
	panic("implement me")
}
