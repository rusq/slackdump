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

package chunk

import (
	"context"
	"errors"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureEncoder struct {
	ch  *Chunk
	err error
}

func (e *captureEncoder) Encode(_ context.Context, ch *Chunk) error {
	e.ch = ch
	return e.err
}

func TestRecorder_CanvasMessages(t *testing.T) {
	t.Run("records roots including an empty final page", func(t *testing.T) {
		enc := &captureEncoder{}
		rec := NewCustomRecorder(enc)
		require.NoError(t, rec.CanvasMessages(t.Context(), "CCANVAS", 2, true, nil))
		require.NotNil(t, enc.ch)
		assert.Equal(t, CCanvasMessages, enc.ch.Type)
		assert.Equal(t, "CCANVAS", enc.ch.ChannelID)
		assert.Equal(t, int32(2), enc.ch.NumThreads)
		assert.Zero(t, enc.ch.Count)
		assert.True(t, enc.ch.IsLast)
	})

	t.Run("preserves encoder errors", func(t *testing.T) {
		want := errors.New("encode")
		rec := NewCustomRecorder(&captureEncoder{err: want})
		assert.ErrorIs(t, rec.CanvasMessages(t.Context(), "CCANVAS", 0, false, nil), want)
	})
}

func TestRecorder_CanvasThreadMessages(t *testing.T) {
	parent := slack.Message{Msg: slack.Msg{Timestamp: "123.456", ThreadTimestamp: "123.456"}}
	replies := []slack.Message{parent, {Msg: slack.Msg{Timestamp: "124.000", ThreadTimestamp: "123.456"}}}

	enc := &captureEncoder{}
	rec := NewCustomRecorder(enc)
	require.NoError(t, rec.CanvasThreadMessages(t.Context(), "CCANVAS", parent, true, replies))
	require.NotNil(t, enc.ch)
	assert.Equal(t, CCanvasThreadMessages, enc.ch.Type)
	assert.Equal(t, "CCANVAS", enc.ch.ChannelID)
	assert.Equal(t, "123.456", enc.ch.ThreadTS)
	assert.Equal(t, &parent, enc.ch.Parent)
	assert.Equal(t, int32(2), enc.ch.Count)
	assert.False(t, enc.ch.ThreadOnly)
	assert.True(t, enc.ch.IsLast)
}
