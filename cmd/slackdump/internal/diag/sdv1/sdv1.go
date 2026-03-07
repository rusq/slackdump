// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package sdv1 contains Slackdump v1.0.x related code.
package sdv1

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/rusq/slack"
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

func load(filepath string) (Messages, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return Messages{}, err
	}
	defer f.Close()
	return readFrom(f)
}

func readFrom(r io.Reader) (Messages, error) {
	var m Messages
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return m, err
	}
	return m, nil
}

func (m Messages) allChannels() []slack.Channel {
	channels := make([]slack.Channel, len(m.SD.Channels))
	copy(channels, m.SD.Channels)
	// it might so happen that the dump file has a channel that is not
	// present in the channels list, i.e. if it's a DM or a group.
	var found bool
	for _, ch := range channels {
		if ch.ID == m.ChannelID {
			found = true
			break
		}
	}
	if !found {
		ci, _ := m.ChannelInfo(context.Background(), m.ChannelID)
		channels = append(channels, *ci)
	}
	return channels
}
