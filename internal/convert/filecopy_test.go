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
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rusq/slackdump/v4/source/mock_source"

	"github.com/rusq/fsadapter"
	"github.com/rusq/fsadapter/mocks/mock_fsadapter"
	"github.com/rusq/slack"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/source"
)

func Test_copy2trg(t *testing.T) {
	t.Run("copy ok", func(t *testing.T) {
		srcdir := t.TempDir()
		trgdir := t.TempDir()

		if err := os.WriteFile(filepath.Join(srcdir, "test.txt"), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
		trgfs := fsadapter.NewDirectory(trgdir)
		srcfs := os.DirFS(srcdir)

		if err := copy2trg(trgfs, "test-copy.txt", srcfs, "test.txt"); err != nil {
			t.Fatal(err)
		}
		// validate
		data, err := os.ReadFile(filepath.Join(trgdir, "test-copy.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "test" {
			t.Fatal("unexpected data")
		}
	})
	t.Run("copy fails", func(t *testing.T) {
		srcdir := t.TempDir()
		trgdir := t.TempDir()

		srcfs := os.DirFS(srcdir)
		trgfs := fsadapter.NewDirectory(trgdir)
		// source file does not exist.
		if err := copy2trg(trgfs, "test-copy.txt", srcfs, "test.txt"); err == nil {
			t.Fatal("expected error, but got nil")
		}
	})
}

func Test_avatarcopywrapper_copyAvatar(t *testing.T) {
	testUser := slack.User{
		ID:      "U12345678",
		Profile: slack.UserProfile{ImageOriginal: "https://example.com/avatar.jpg"},
	}
	type args struct {
		u slack.User
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(mfsa *mock_fsadapter.MockFSCloser, mavst *mock_source.MockStorage)
		wantErr  bool
	}{
		{
			name: "copies avatar",
			args: args{u: testUser},
			expectFn: func(mfsa *mock_fsadapter.MockFSCloser, mavst *mock_source.MockStorage) {
				tmpfile, err := os.CreateTemp(t.TempDir(), "")
				if err != nil {
					t.Fatal(err)
				}

				mockfs := fstest.MapFS{
					chunk.AvatarsDir + "/U12345678/avatar.jpg": {
						Data: []byte("avatar"),
					},
				}

				mavst.EXPECT().FS().Return(mockfs).Times(1)
				const avpath = chunk.AvatarsDir + "/U12345678/avatar.jpg"
				mavst.EXPECT().File(gomock.Any(), gomock.Any()).Return(avpath, nil).Times(1)
				mfsa.EXPECT().Create(avpath).Return(tmpfile, nil).Times(1)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mfsa := mock_fsadapter.NewMockFSCloser(ctrl)
			mavst := mock_source.NewMockStorage(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mfsa, mavst)
			}
			a := &avatarcopywrapper{
				fsa:  mfsa,
				avst: mavst,
			}
			if err := a.copyAvatar(tt.args.u); (err != nil) != tt.wantErr {
				t.Errorf("avatarcopywrapper.copyAvatar() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileCopier_Copy(t *testing.T) {
	channel := &slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{ID: "C123"},
		},
	}

	t.Run("returns nil for skipped files", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		st := mock_source.NewMockStorage(ctrl)

		src.EXPECT().Files().Return(st).Times(1)
		st.EXPECT().FS().Return(fstest.MapFS{}).Times(1)

		c := NewFileCopier(src, fsadapter.NewDirectory(t.TempDir()), source.MattermostFilepath, true)
		msg := &slack.Message{Msg: slack.Msg{
			Timestamp: "123.456",
			Files: []slack.File{
				{ID: "F1", Mode: "tombstone", Name: "gone.txt"},
				{ID: "F2", Mode: "hidden_by_limit", Name: "hidden.txt"},
				{ID: "F3", Mode: "external", Name: "external.txt", IsExternal: true},
			},
		}}

		if err := c.Copy(channel, msg); err != nil {
			t.Fatalf("Copy() error = %v, want nil", err)
		}
	})

	t.Run("returns error when source path cannot be resolved", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		st := mock_source.NewMockStorage(ctrl)

		src.EXPECT().Files().Return(st).Times(2)
		st.EXPECT().FS().Return(fstest.MapFS{}).Times(1)
		st.EXPECT().File("F1", "missing.txt").Return("", fs.ErrNotExist).Times(1)

		c := NewFileCopier(src, fsadapter.NewDirectory(t.TempDir()), source.MattermostFilepath, true)
		msg := &slack.Message{Msg: slack.Msg{
			Timestamp: "123.456",
			Files:     []slack.File{{ID: "F1", Name: "missing.txt"}},
		}}

		err := c.Copy(channel, msg)
		if err == nil || !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("Copy() error = %v, want fs.ErrNotExist", err)
		}
	})

	t.Run("returns error when resolved source file cannot be statted", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		st := mock_source.NewMockStorage(ctrl)

		src.EXPECT().Files().Return(st).Times(2)
		st.EXPECT().FS().Return(fstest.MapFS{}).Times(1)
		st.EXPECT().File("F1", "missing.txt").Return("uploads/F1/missing.txt", nil).Times(1)

		c := NewFileCopier(src, fsadapter.NewDirectory(t.TempDir()), source.MattermostFilepath, true)
		msg := &slack.Message{Msg: slack.Msg{
			Timestamp: "123.456",
			Files:     []slack.File{{ID: "F1", Name: "missing.txt"}},
		}}

		err := c.Copy(channel, msg)
		if err == nil || !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("Copy() error = %v, want fs.ErrNotExist", err)
		}
	})

	t.Run("returns error when target create fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		src := mock_source.NewMockSourcer(ctrl)
		st := mock_source.NewMockStorage(ctrl)
		trg := mock_fsadapter.NewMockFSCloser(ctrl)

		src.EXPECT().Files().Return(st).Times(2)
		st.EXPECT().FS().Return(fstest.MapFS{
			"uploads/F1/ok.txt": {Data: []byte("content")},
		}).Times(1)
		st.EXPECT().File("F1", "ok.txt").Return("uploads/F1/ok.txt", nil).Times(1)
		trg.EXPECT().Create(source.MattermostFilepath(channel, &slack.File{ID: "F1", Name: "ok.txt"})).Return(nil, errors.New("create failed")).Times(1)

		c := NewFileCopier(src, trg, source.MattermostFilepath, true)
		msg := &slack.Message{Msg: slack.Msg{
			Timestamp: "123.456",
			Files:     []slack.File{{ID: "F1", Name: "ok.txt"}},
		}}

		err := c.Copy(channel, msg)
		if err == nil || !strings.Contains(err.Error(), "create failed") {
			t.Fatalf("Copy() error = %v, want create failed", err)
		}
	})
}
