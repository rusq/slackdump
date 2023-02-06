package processors

import "github.com/slack-go/slack"

type Discarder struct{}

func (d *Discarder) Messages(messages []slack.Message) error {
	return nil
}

func (d *Discarder) ThreadMessages(parent slack.Message, replies []slack.Message) error {
	return nil
}

func (d *Discarder) Files(parent slack.Message, files []slack.File) error {
	return nil
}
