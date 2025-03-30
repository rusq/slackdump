// Package sdv1 contains Slackdump v1.0.x related code.
package sdv1

import (
	"encoding/json"
	"io"
	"os"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/types"
)

// Messages keeps the slice of messages.
type Messages struct {
	Messages  []slack.Message
	ChannelID string
	SD        State
}

type State struct {
	Users    Users           `json:"users"`
	Channels []slack.Channel `json:"channels"`
}

type Users struct {
	Users []slack.User
}

func Load(filepath string) (Messages, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return Messages{}, err
	}
	defer f.Close()
	return ReadFrom(f)
}

func ReadFrom(r io.Reader) (Messages, error) {
	var m Messages
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return m, err
	}
	return m, nil
}

func (m Messages) Conversation() types.Conversation {
	// find this channel name
	var name string
	for _, c := range m.SD.Channels {
		if c.ID == m.ChannelID {
			name = c.Name
			break
		}
	}
	mm := make([]types.Message, len(m.Messages))
	for i, msg := range m.Messages {
		mm[i] = types.Message{
			Message: msg,
		}
		mm[i].Blocks = slack.Blocks{} // ignore blocks, they are damaged in v1.0.x dumps
	}

	return types.Conversation{
		ID:       m.ChannelID,
		Messages: mm,
		Name:     name,
	}
}

func (m Messages) ChannelInfo() *slack.Channel {
	var ci *slack.Channel
	for _, ch := range m.SD.Channels {
		if ch.ID == m.ChannelID {
			ci = &ch
			break
		}
	}
	if ci == nil {
		ci = structures.ChannelFromID(m.ChannelID) // craft a fake one
	}
	switch m.ChannelID[0] {
	case 'D':
		ci.IsIM = true
	case 'G':
		ci.IsMpIM = true
	case 'C':
		ci.IsChannel = true
	}
	users := make(map[string]struct{})
	for _, m := range m.Messages {
		if m.User != "" {
			if _, ok := users[m.User]; !ok {
				users[m.User] = struct{}{}
				ci.Members = append(ci.Members, m.User)
			}
		}
	}
	return ci
}

func (m Messages) Msgs() []slack.Message {
	mm := make([]slack.Message, len(m.Messages))
	copy(mm, m.Messages)
	for i := range m.Messages {
		mm[i].Blocks = slack.Blocks{} // delete blocks, they are damaged in v1.0.x dumps
	}
	return mm
}
