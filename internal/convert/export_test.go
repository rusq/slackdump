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

package convert

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/source"
	"github.com/rusq/slackdump/v4/source/mock_source"
)

const (
	testSrcDir = "../../tmp/ora600" // TODO: fix manual nature of this/obfuscate
)

var testLogger = slog.Default()

func TestChunkToExport_Validate(t *testing.T) {
	fixtures.SkipInCI(t)
	fixtures.SkipIfNotExist(t, testSrcDir)
	srcDir, err := chunk.OpenDir(testSrcDir)
	if err != nil {
		t.Fatal(err)
	}
	defer srcDir.Close()
	src := source.OpenChunkDir(srcDir, true)
	testTrgDir := t.TempDir()

	type fields struct {
		Src       source.Sourcer
		Trg       fsadapter.FS
		opts      options
		UploadDir string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"empty", fields{}, true},
		{"no source", fields{Trg: fsadapter.NewDirectory(testTrgDir)}, true},
		{"no target", fields{Src: src}, true},
		{
			"valid, no files",
			fields{
				Src: src,
				Trg: fsadapter.NewDirectory(testTrgDir),
				opts: options{
					includeFiles: false,
				},
			},
			false,
		},
		{
			"valid, include files, but no location functions",
			fields{
				Src: src,
				Trg: fsadapter.NewDirectory(testTrgDir),
				opts: options{
					includeFiles: true,
				},
			},
			true,
		},
		{
			"valid, include files, with location functions",
			fields{
				Src: src,
				Trg: fsadapter.NewDirectory(testTrgDir),
				opts: options{
					includeFiles: true,
					trgFileLoc: func(*slack.Channel, *slack.File) string {
						return ""
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ToExport{
				src:  tt.fields.Src,
				trg:  tt.fields.Trg,
				opts: tt.fields.opts,
			}
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ChunkToExport.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChunkToExport_Convert(t *testing.T) {
	setupMockSource := func(t *testing.T, msg slack.Message, st *mock_source.MockStorage) *mock_source.MockSourcer {
		t.Helper()

		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		channel := &slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{ID: "C123"},
				Name:         "general",
			},
		}

		src.EXPECT().Users(gomock.Any()).Return([]slack.User{{ID: "U1", Name: "tester"}}, nil).AnyTimes()
		src.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{*channel}, nil).AnyTimes()
		src.EXPECT().WorkspaceInfo(gomock.Any()).Return(&slack.AuthTestResponse{
			UserID: "U1",
			TeamID: "T1",
			URL:    "https://example.slack.com/",
		}, nil).AnyTimes()
		src.EXPECT().Files().Return(st).AnyTimes()
		src.EXPECT().Name().Return("mock-source").AnyTimes()
		src.EXPECT().ChannelInfo(gomock.Any(), "C123").Return(channel, nil)
		src.EXPECT().Sorted(gomock.Any(), "C123", false, gomock.Any()).DoAndReturn(
			func(_ context.Context, _ string, _ bool, cb func(time.Time, *slack.Message) error) error {
				return cb(time.Unix(1710000000, 0), &msg)
			},
		)

		return src
	}

	t.Run("skipped file modes do not fail conversion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		st := mock_source.NewMockStorage(ctrl)
		st.EXPECT().Type().Return(source.STmattermost).AnyTimes()
		st.EXPECT().FS().Return(fstest.MapFS{}).Times(1)

		msg := slack.Message{Msg: slack.Msg{
			Timestamp: "1710000000.000001",
			Text:      "message",
			Files: []slack.File{{
				ID:   "F1",
				Name: "gone.txt",
				Mode: "tombstone",
			}},
		}}
		src := setupMockSource(t, msg, st)

		c := NewToExport(src, fsadapter.NewDirectory(t.TempDir()), WithIncludeFiles(true))
		c.workers = 1
		c.opts.lg = testLogger
		if err := c.Convert(t.Context()); err != nil {
			t.Fatalf("Convert() error = %v, want nil", err)
		}
	})

	t.Run("real copy failures still fail conversion", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		st := mock_source.NewMockStorage(ctrl)
		st.EXPECT().Type().Return(source.STmattermost).AnyTimes()
		st.EXPECT().FS().Return(fstest.MapFS{}).Times(1)
		st.EXPECT().File("F1", "missing.txt").Return("", os.ErrNotExist).Times(1)

		msg := slack.Message{Msg: slack.Msg{
			Timestamp: "1710000000.000001",
			Text:      "message",
			Files: []slack.File{{
				ID:   "F1",
				Name: "missing.txt",
			}},
		}}
		src := setupMockSource(t, msg, st)

		c := NewToExport(src, fsadapter.NewDirectory(t.TempDir()), WithIncludeFiles(true))
		c.workers = 1
		c.opts.lg = testLogger
		err := c.Convert(t.Context())
		if err == nil || err.Error() != "convert: there were errors" {
			t.Fatalf("Convert() error = %v, want convert: there were errors", err)
		}
	})

	t.Run("integration", func(t *testing.T) {
		fixtures.SkipInCI(t)
		fixtures.SkipIfNotExist(t, testSrcDir)
		cd, err := chunk.OpenDir(testSrcDir)
		if err != nil {
			t.Fatal(err)
		}
		defer cd.Close()
		testTrgDir, err := os.MkdirTemp("", "slackdump")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(testTrgDir)
		fsa, err := fsadapter.NewZipFile(filepath.Join(testTrgDir, "slackdump.zip"))
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()
		src := source.OpenChunkDir(cd, true)
		c := NewToExport(src, fsa, WithIncludeFiles(true))

		ctx := t.Context()
		c.opts.lg = testLogger
		if err := c.Convert(ctx); err != nil {
			t.Fatal(err)
		}
	})
}
