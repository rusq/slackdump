package state

import "github.com/slack-go/slack"

type Conversation struct {
	LastMsg       *slack.Message
	LastThreadMsg *slack.Message
	LastFile      *slack.File
}

type Channel struct {
	ChannelID string
}

type State struct {
	LastChannel   Channel
	Conversations map[string]Conversation
}

func Save(state State) error {

	return nil
}
