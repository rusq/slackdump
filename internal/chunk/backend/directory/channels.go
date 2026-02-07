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
package directory

import (
	"context"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

// Channels is a processor that writes the channel information into the
// channels file.
type Channels struct {
	*dirproc
}

// NewChannels creates a new Channels processor.  fn is called for each
// channel chunk that is retrieved.  The function is called before the chunk
// is processed by the recorder.
func NewChannels(dir *chunk.Directory) (*Channels, error) {
	p, err := newDirProc(dir, chunk.FChannels)
	if err != nil {
		return nil, err
	}
	return &Channels{dirproc: p}, nil
}

// Channels is called for each channel chunk that is retrieved.  Then, the
// function calls the function passed in to the constructor for the channel
// slice.
func (cp *Channels) Channels(ctx context.Context, channels []slack.Channel) error {
	if err := cp.dirproc.Channels(ctx, channels); err != nil {
		return err
	}
	return nil
}
