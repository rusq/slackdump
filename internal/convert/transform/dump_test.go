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

package transform

import (
	"context"
	"iter"
	"path/filepath"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4/source"
	"github.com/stretchr/testify/require"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/internal/nametmpl"
)

type canvasDumpSource struct {
	source.Sourcer
	owner  *slack.Channel
	roots  []slack.Message
	thread []slack.Message
}

func (s canvasDumpSource) ChannelInfo(context.Context, string) (*slack.Channel, error) {
	return s.owner, nil
}

func (s canvasDumpSource) AllMessages(context.Context, string) (iter.Seq2[slack.Message, error], error) {
	return messageIter(nil), nil
}

func (s canvasDumpSource) CanvasMessages(context.Context, string) (iter.Seq2[slack.Message, error], error) {
	return messageIter(s.roots), nil
}

func (s canvasDumpSource) CanvasThreadMessages(context.Context, string, string) (iter.Seq2[slack.Message, error], error) {
	return messageIter(s.thread), nil
}

func messageIter(messages []slack.Message) iter.Seq2[slack.Message, error] {
	return func(yield func(slack.Message, error) bool) {
		for _, m := range messages {
			if !yield(m, nil) {
				return
			}
		}
	}
}

func Test_stdConvert(t *testing.T) {
	testNames := []chunk.FileID{
		"CHYLGDP0D-1682335799.257359",
		"CHYLGDP0D-1682375167.836499",
		"CHM82GF99",
	}
	t.Run("manual", func(t *testing.T) {
		testDir := filepath.Join("..", "..", "..", "tmp", "3")
		fixtures.SkipIfNotExist(t, testDir)

		ctx := t.Context()

		src, err := source.Load(ctx, testDir)
		if err != nil {
			t.Fatal(err)
		}
		defer src.Close()
		tmp := t.TempDir()
		fsa, err := fsadapter.New(filepath.Join(tmp, "output-dump.zip"))
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()
		cvt := DumpConverter{
			src:  src,
			fsa:  fsa,
			tmpl: nametmpl.NewDefault(),
		}

		for i, name := range testNames {
			id, thread := chunk.FileID(name).Split()
			if err := cvt.Convert(ctx, id, thread); err != nil {
				t.Fatalf("failed on i=%d, name=%s: %s", i, name, err)
			}
		}
	})
}

func TestDumpConverter_convertCanvas(t *testing.T) {
	owner := &slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{
				ID: "COWNER",
			},
			Name: "owner",
		},
		Properties: &slack.Properties{Canvas: slack.Canvas{FileId: "FCANVAS"}},
	}
	root := slack.Message{Msg: slack.Msg{
		Timestamp:       "1700000000.000001",
		ThreadTimestamp: "1700000000.000001",
		ReplyCount:      1,
		Text:            "root",
	}}
	reply := slack.Message{Msg: slack.Msg{
		Timestamp:       "1700000001.000001",
		ThreadTimestamp: root.Timestamp,
		Text:            "reply",
	}}
	src := canvasDumpSource{
		owner:  owner,
		roots:  []slack.Message{root},
		thread: []slack.Message{root, reply},
	}
	for _, tt := range []struct {
		name   string
		output func(t *testing.T) string
	}{
		{name: "directory", output: func(t *testing.T) string { return t.TempDir() }},
		{name: "zip", output: func(t *testing.T) string { return filepath.Join(t.TempDir(), "canvas.zip") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.output(t)
			fsa, err := fsadapter.New(output)
			require.NoError(t, err)
			cvt, err := NewDump(fsa, src)
			require.NoError(t, err)
			require.NoError(t, cvt.Convert(t.Context(), owner.ID, ""))
			require.NoError(t, fsa.Close())

			loaded, err := source.Load(t.Context(), output)
			require.NoError(t, err)
			defer loaded.Close()
			canvas, ok := loaded.(canvasSource)
			require.True(t, ok)
			it, err := canvas.CanvasThreadMessages(t.Context(), "CCANVAS", root.Timestamp)
			require.NoError(t, err)
			var got []slack.Message
			for m, err := range it {
				require.NoError(t, err)
				got = append(got, m)
			}
			require.Len(t, got, 2)
			require.Equal(t, "reply", got[1].Text)
		})
	}
}
