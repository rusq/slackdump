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

package processor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/processor"
)

type testConversations struct{}

func (*testConversations) Messages(context.Context, string, int, bool, []slack.Message) error {
	return nil
}

func (*testConversations) ThreadMessages(context.Context, string, slack.Message, bool, bool, []slack.Message) error {
	return nil
}

func (*testConversations) Files(context.Context, *slack.Channel, slack.Message, []slack.File) error {
	return nil
}

func (*testConversations) ChannelInfo(context.Context, *slack.Channel, string) error {
	return nil
}

func (*testConversations) ChannelUsers(context.Context, string, string, []string) error {
	return nil
}

func (*testConversations) Close() error {
	return nil
}

type testCanvasMessenger struct {
	testConversations
	name  string
	calls *[]string
	err   error
}

func (m *testCanvasMessenger) CanvasMessages(context.Context, string, int, bool, []slack.Message) error {
	*m.calls = append(*m.calls, m.name+":roots")
	return m.err
}

func (m *testCanvasMessenger) CanvasThreadMessages(context.Context, string, slack.Message, bool, []slack.Message) error {
	*m.calls = append(*m.calls, m.name+":thread")
	return m.err
}

func TestAsCanvasMessenger(t *testing.T) {
	var calls []string
	canvas := &testCanvasMessenger{name: "core", calls: &calls}

	tests := []struct {
		name string
		proc processor.Conversations
		want bool
	}{
		{
			name: "direct capability",
			proc: canvas,
			want: true,
		},
		{
			name: "nested wrappers preserve capability",
			proc: processor.AppendMessenger(
				processor.PrependMessenger(canvas, &testConversations{}),
				&testConversations{},
			),
			want: true,
		},
		{
			name: "wrapped legacy processor remains unsupported",
			proc: processor.AppendMessenger(
				processor.PrependMessenger(&testConversations{}, &testConversations{}),
				&testConversations{},
			),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := processor.AsCanvasMessenger(tt.proc)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestJointConversations_CanvasMessages(t *testing.T) {
	var calls []string
	beforeErr := errors.New("before")
	coreErr := errors.New("core")
	afterErr := errors.New("after")
	proc := processor.AppendMessenger(
		processor.PrependMessenger(
			&testCanvasMessenger{name: "core", calls: &calls, err: coreErr},
			&testCanvasMessenger{name: "before", calls: &calls, err: beforeErr},
		),
		&testCanvasMessenger{name: "after", calls: &calls, err: afterErr},
	)
	cm, ok := processor.AsCanvasMessenger(proc)
	require.True(t, ok)

	err := cm.CanvasMessages(t.Context(), "C123", 1, true, []slack.Message{{}})

	assert.Equal(t, []string{"before:roots", "core:roots", "after:roots"}, calls)
	assert.ErrorIs(t, err, beforeErr)
	assert.ErrorIs(t, err, coreErr)
	assert.ErrorIs(t, err, afterErr)
}

func TestJointConversations_CanvasThreadMessages(t *testing.T) {
	var calls []string
	beforeErr := errors.New("before")
	coreErr := errors.New("core")
	afterErr := errors.New("after")
	proc := processor.AppendMessenger(
		processor.PrependMessenger(
			&testCanvasMessenger{name: "core", calls: &calls, err: coreErr},
			&testCanvasMessenger{name: "before", calls: &calls, err: beforeErr},
		),
		&testCanvasMessenger{name: "after", calls: &calls, err: afterErr},
	)
	cm, ok := processor.AsCanvasMessenger(proc)
	require.True(t, ok)

	err := cm.CanvasThreadMessages(t.Context(), "C123", slack.Message{}, true, []slack.Message{{}})

	assert.Equal(t, []string{"before:thread", "core:thread", "after:thread"}, calls)
	assert.ErrorIs(t, err, beforeErr)
	assert.ErrorIs(t, err, coreErr)
	assert.ErrorIs(t, err, afterErr)
}
