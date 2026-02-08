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
package redownload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"iter"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/source"
	"github.com/rusq/slackdump/v4/source/mock_source"
	"go.uber.org/mock/gomock"
)

type stubStorage struct {
	typ    source.StorageType
	pathFn func(*slack.Channel, *slack.File) string
}

func (s stubStorage) FS() fs.FS                           { return nil }
func (s stubStorage) Type() source.StorageType            { return s.typ }
func (s stubStorage) File(string, string) (string, error) { return "", fs.ErrNotExist }
func (s stubStorage) FilePath(ch *slack.Channel, f *slack.File) string {
	if s.pathFn != nil {
		return s.pathFn(ch, f)
	}
	return ""
}

type stubSource struct {
	name           string
	storage        source.Storage
	channels       []slack.Channel
	channelsErr    error
	messages       func(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error)
	threadMessages func(ctx context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error)
	allMessagesErr error
	allThreadsErr  error
}

func (s stubSource) Name() string { return s.name }
func (s stubSource) Type() source.Flags {
	return 0
}
func (s stubSource) Channels(context.Context) ([]slack.Channel, error) {
	if s.channelsErr != nil {
		return nil, s.channelsErr
	}
	return s.channels, nil
}
func (s stubSource) Users(context.Context) ([]slack.User, error) { return nil, nil }
func (s stubSource) AllMessages(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	if s.messages != nil {
		return s.messages(ctx, channelID)
	}
	if s.allMessagesErr != nil {
		return nil, s.allMessagesErr
	}
	return nil, nil
}
func (s stubSource) AllThreadMessages(ctx context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	if s.threadMessages != nil {
		return s.threadMessages(ctx, channelID, threadID)
	}
	if s.allThreadsErr != nil {
		return nil, s.allThreadsErr
	}
	return nil, nil
}
func (s stubSource) Sorted(context.Context, string, bool, func(time.Time, *slack.Message) error) error {
	return nil
}
func (s stubSource) ChannelInfo(context.Context, string) (*slack.Channel, error) { return nil, nil }
func (s stubSource) Files() source.Storage                                       { return s.storage }
func (s stubSource) Avatars() source.Storage                                     { return nil }
func (s stubSource) WorkspaceInfo(context.Context) (*slack.AuthTestResponse, error) {
	return nil, nil
}
func (s stubSource) Latest(context.Context) (map[structures.SlackLink]time.Time, error) {
	return nil, nil
}
func (s stubSource) Close() error { return nil }

type recordingDownloader struct {
	paths []string
}

func (d *recordingDownloader) Download(fullpath, _ string) error {
	d.paths = append(d.paths, fullpath)
	return nil
}
func (d *recordingDownloader) Stop() {}

func seqFromMessages(msgs []slack.Message) iter.Seq2[slack.Message, error] {
	return func(yield func(slack.Message, error) bool) {
		for _, m := range msgs {
			if !yield(m, nil) {
				return
			}
		}
	}
}

func Test_pathFunc(t *testing.T) {
	ch := &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C1"}}}
	file := &slack.File{ID: "F1", Name: "file.txt"}

	t.Run("storage path used when available", func(t *testing.T) {
		st := stubStorage{
			typ: source.STstandard,
			pathFn: func(*slack.Channel, *slack.File) string {
				return "custom/path"
			},
		}
		r := &Redownloader{src: stubSource{name: "/tmp", storage: st}}

		got := r.pathFunc()(ch, file)
		if got != "custom/path" {
			t.Fatalf("pathFunc() = %q, want %q", got, "custom/path")
		}
	})

	t.Run("dump fallback when no storage", func(t *testing.T) {
		r := &Redownloader{flags: source.FDump, src: stubSource{name: "/tmp", storage: stubStorage{typ: source.STnone}}}

		got := r.pathFunc()(ch, file)
		want := source.DumpFilepath(ch, file)
		if got != want {
			t.Fatalf("pathFunc() = %q, want %q", got, want)
		}
	})

	t.Run("mattermost fallback by default", func(t *testing.T) {
		r := &Redownloader{src: stubSource{name: "/tmp", storage: stubStorage{typ: source.STnone}}}

		got := r.pathFunc()(ch, file)
		want := source.MattermostFilepath(ch, file)
		if got != want {
			t.Fatalf("pathFunc() = %q, want %q", got, want)
		}
	})
}

func Test_fileProc(t *testing.T) {
	ctx := context.Background()
	ch := &slack.Channel{GroupConversation: slack.GroupConversation{
		Conversation: slack.Conversation{ID: "C123"},
		Name:         "chan",
	}}
	f := slack.File{ID: "F1", Name: "file.txt"}

	tests := []struct {
		name      string
		flags     source.Flags
		storage   stubStorage
		wantPath  func(*slack.Channel, *slack.File) string
		wantError bool
	}{
		{
			name:     "database uses mattermost path",
			flags:    source.FDatabase,
			storage:  stubStorage{typ: source.STnone},
			wantPath: source.MattermostFilepath,
		},
		{
			name:     "export with storage uses its path func",
			flags:    source.FExport,
			storage:  stubStorage{typ: source.STmattermost},
			wantPath: source.MattermostFilepath,
		},
		{
			name:     "export defaults to mattermost when storage unknown",
			flags:    source.FExport,
			storage:  stubStorage{typ: source.STnone},
			wantPath: source.MattermostFilepath,
		},
		{
			name:     "dump uses dump filepath",
			flags:    source.FDump,
			storage:  stubStorage{typ: source.STnone},
			wantPath: source.DumpFilepath,
		},
		{
			name:      "unsupported flags error",
			flags:     source.FUnknown,
			storage:   stubStorage{typ: source.STnone},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := &recordingDownloader{}
			r := &Redownloader{
				flags: tt.flags,
				src:   stubSource{storage: tt.storage},
			}

			fp, err := r.fileProc(dl)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("fileProc() error = %v", err)
			}

			err = fp.Files(ctx, ch, slack.Message{}, []slack.File{f})
			if err != nil {
				t.Fatalf("Files() error = %v", err)
			}

			if len(dl.paths) != 1 {
				t.Fatalf("download paths = %v, want 1 entry", dl.paths)
			}

			got := dl.paths[0]
			want := tt.wantPath(ch, &f)
			if got != want {
				t.Fatalf("download path = %q, want %q", got, want)
			}
		})
	}
}

func Test_channels(t *testing.T) {
	t.Run("returns channels", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockSrc := mock_source.NewMockSourcer(ctrl)
		mockSrc.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C1"}}}}, nil)
		r := &Redownloader{src: mockSourceResumeCloser{MockSourcer: mockSrc}}

		chans, err := r.channels(context.Background())
		if err != nil {
			t.Fatalf("channels() error = %v", err)
		}
		if len(chans) != 1 || chans[0].ID != "C1" {
			t.Fatalf("channels() = %#v, want one channel with ID C1", chans)
		}
	})

	t.Run("no channels error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockSrc := mock_source.NewMockSourcer(ctrl)
		mockSrc.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{}, nil)
		r := &Redownloader{src: mockSourceResumeCloser{MockSourcer: mockSrc}}

		_, err := r.channels(context.Background())
		if !errors.Is(err, ErrNoChannels) {
			t.Fatalf("channels() error = %v, want %v", err, ErrNoChannels)
		}
	})

	t.Run("propagates error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockSrc := mock_source.NewMockSourcer(ctrl)
		mockSrc.EXPECT().Channels(gomock.Any()).Return(nil, errors.New("boom"))
		r := &Redownloader{src: mockSourceResumeCloser{MockSourcer: mockSrc}}

		_, err := r.channels(context.Background())
		if err == nil || err.Error() != "error reading channels: boom" {
			t.Fatalf("channels() error = %v, want wrapped boom", err)
		}
	})
}

type mockSourceResumeCloser struct {
	*mock_source.MockSourcer
}

func (m mockSourceResumeCloser) Latest(context.Context) (map[structures.SlackLink]time.Time, error) {
	return nil, nil
}
func (m mockSourceResumeCloser) Close() error { return nil }

func Test_scanChannel(t *testing.T) {
	ch := slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C1"}}}

	t.Run("returns nil on not found", func(t *testing.T) {
		r := &Redownloader{
			src: stubSource{
				allMessagesErr: source.ErrNotFound,
				storage:        stubStorage{typ: source.STnone},
			},
			lg: slog.Default(),
		}
		items, err := r.scanChannel(context.Background(), &ch)
		if err != nil {
			t.Fatalf("scanChannel() error = %v", err)
		}
		if items != nil {
			t.Fatalf("scanChannel() = %#v, want nil", items)
		}
	})

	t.Run("wraps message errors", func(t *testing.T) {
		r := &Redownloader{
			src: stubSource{
				allMessagesErr: errors.New("boom"),
				storage:        stubStorage{typ: source.STnone},
			},
			lg: slog.Default(),
		}
		_, err := r.scanChannel(context.Background(), &ch)
		if err == nil || err.Error() != "error reading messages: boom" {
			t.Fatalf("scanChannel() error = %v, want wrapped boom", err)
		}
	})

	t.Run("collects missing files", func(t *testing.T) {
		tmp := t.TempDir()
		storage := stubStorage{
			typ: source.STstandard,
			pathFn: func(*slack.Channel, *slack.File) string {
				return "missing-file"
			},
		}
		msg := slack.Message{Msg: slack.Msg{
			Files: []slack.File{
				{ID: "F1", Name: "file.txt", Size: 10},
			},
		}}
		r := &Redownloader{
			src: stubSource{
				name:    tmp,
				storage: storage,
				messages: func(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
					return seqFromMessages([]slack.Message{msg}), nil
				},
			},
			lg: slog.Default(),
		}

		items, err := r.scanChannel(context.Background(), &ch)
		if err != nil {
			t.Fatalf("scanChannel() error = %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("scanChannel() = %#v, want 1 item", items)
		}
		if items[0].f.ID != "F1" {
			t.Fatalf("unexpected file id %q", items[0].f.ID)
		}
	})
}

func Test_scanMsgs_filePresence(t *testing.T) {
	tmp := t.TempDir()
	ch := slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C1"}}}
	file := slack.File{ID: "F1", Name: "file.txt", Size: 10}
	pathFn := func(*slack.Channel, *slack.File) string {
		return "present-file"
	}
	storage := stubStorage{typ: source.STstandard, pathFn: pathFn}

	r := &Redownloader{
		src: stubSource{
			name:    tmp,
			storage: storage,
		},
		lg: slog.Default(),
	}

	// Create a present file to ensure it is skipped.
	presentPath := filepath.Join(tmp, pathFn(&ch, &file))
	if err := os.MkdirAll(filepath.Dir(presentPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(presentPath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	msgs := []slack.Message{
		{Msg: slack.Msg{Files: []slack.File{file}}},
	}

	items, err := r.scanMsgs(context.Background(), &ch, seqFromMessages(msgs), false)
	if err != nil {
		t.Fatalf("scanMsgs() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("scanMsgs() = %#v, want none (file exists)", items)
	}

	t.Run("zero length treated missing", func(t *testing.T) {
		if err := os.WriteFile(presentPath, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		items, err := r.scanMsgs(context.Background(), &ch, seqFromMessages(msgs), false)
		if err != nil {
			t.Fatalf("scanMsgs() error = %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("scanMsgs() = %#v, want 1 item for zero-length file", items)
		}
	})
}

func Test_scanMsgs_threads(t *testing.T) {
	tmp := t.TempDir()
	ch := slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C1"}}}
	pathFn := func(_ *slack.Channel, _ *slack.File) string { return "thread-file" }
	storage := stubStorage{typ: source.STstandard, pathFn: pathFn}

	threadMsg := slack.Message{
		Msg: slack.Msg{
			Timestamp:       "1.0",
			ThreadTimestamp: "1.0",
			LatestReply:     "1.1",
		},
	}
	threadFile := slack.File{ID: "F2", Name: "thread.txt", Size: 5}

	r := &Redownloader{
		src: stubSource{
			name:    tmp,
			storage: storage,
			threadMessages: func(ctx context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
				return seqFromMessages([]slack.Message{{Msg: slack.Msg{Timestamp: "1.1", Files: []slack.File{threadFile}}}}), nil
			},
		},
		lg: slog.Default(),
	}

	items, err := r.scanMsgs(context.Background(), &ch, seqFromMessages([]slack.Message{threadMsg}), false)
	if err != nil {
		t.Fatalf("scanMsgs() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("scanMsgs() = %#v, want 1 item from thread", items)
	}
	if items[0].f.ID != threadFile.ID {
		t.Fatalf("unexpected file id %q, want %q", items[0].f.ID, threadFile.ID)
	}
}

func Test_processChannel(t *testing.T) {
	ch := slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C1"}}}
	file := slack.File{ID: "F1", Name: "file.txt", Size: 42}

	t.Run("counts files and bytes, calls callback", func(t *testing.T) {
		tmp := t.TempDir()
		storage := stubStorage{typ: source.STstandard, pathFn: func(*slack.Channel, *slack.File) string { return "missing" }}
		r := &Redownloader{
			src: stubSource{
				name:    tmp,
				storage: storage,
				messages: func(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
					return seqFromMessages([]slack.Message{{Msg: slack.Msg{Files: []slack.File{file}}}}), nil
				},
			},
			lg: slog.Default(),
		}

		var called int
		stats, err := r.processChannel(context.Background(), &ch, func(item *dlItem) error {
			called++
			if item.f.ID != file.ID {
				t.Fatalf("callback file id = %q, want %q", item.f.ID, file.ID)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("processChannel() error = %v", err)
		}
		if called != 1 {
			t.Fatalf("callback called %d times, want 1", called)
		}
		if stats.NumFiles != 1 || stats.NumBytes != uint64(file.Size) {
			t.Fatalf("stats = %+v, want files=1 bytes=%d", stats, file.Size)
		}
	})

	t.Run("callback error stops processing", func(t *testing.T) {
		tmp := t.TempDir()
		storage := stubStorage{typ: source.STstandard, pathFn: func(*slack.Channel, *slack.File) string { return "missing" }}
		r := &Redownloader{
			src: stubSource{
				name:    tmp,
				storage: storage,
				messages: func(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
					return seqFromMessages([]slack.Message{{Msg: slack.Msg{Files: []slack.File{file}}}}), nil
				},
			},
			lg: slog.Default(),
		}

		stats, err := r.processChannel(context.Background(), &ch, func(item *dlItem) error {
			return errors.New("boom")
		})
		if err == nil || err.Error() != "boom" {
			t.Fatalf("processChannel() error = %v, want boom", err)
		}
		if stats.NumFiles != 0 || stats.NumBytes != 0 {
			t.Fatalf("stats = %+v, want zeroed on error", stats)
		}
	})
}

func Test_validate(t *testing.T) {
	t.Run("zip sources rejected", func(t *testing.T) {
		// We cannot reset the global exit status, but ensure it reaches at least SUserError.
		err := validate(source.FZip)
		if err == nil || !strings.Contains(err.Error(), "unpack it first") {
			t.Fatalf("validate(FZip) error = %v, want zip error", err)
		}
	})

	t.Run("non-zip sources allowed", func(t *testing.T) {
		if err := validate(source.FDump); err != nil {
			t.Fatalf("validate(FDump) error = %v, want nil", err)
		}
	})
}

type fileGetterFunc func(ctx context.Context, downloadURL string, writer io.Writer) error

func (f fileGetterFunc) GetFileContext(ctx context.Context, downloadURL string, writer io.Writer) error {
	return f(ctx, downloadURL, writer)
}

func TestDownload(t *testing.T) {
	tmp := t.TempDir()
	ch := slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C1"}}}
	file := slack.File{ID: "F1", Name: "file.txt", URLPrivateDownload: "https://files.slack.test/file1", Size: 5}

	fg := fileGetterFunc(func(ctx context.Context, downloadURL string, w io.Writer) error {
		if downloadURL != file.URLPrivateDownload {
			return fmt.Errorf("unexpected url %q", downloadURL)
		}
		_, err := io.WriteString(w, "hello")
		return err
	})

	r := &Redownloader{
		flags: source.FDump,
		src: stubSource{
			name:    tmp,
			storage: stubStorage{typ: source.STnone},
			channels: []slack.Channel{
				ch,
			},
			messages: func(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
				return seqFromMessages([]slack.Message{{Msg: slack.Msg{Files: []slack.File{file}}}}), nil
			},
		},
		lg: slog.Default(),
	}

	stats, err := r.Download(context.Background(), fg)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if stats.NumFiles != 1 || stats.NumBytes != uint64(file.Size) {
		t.Fatalf("Download() stats = %+v, want files=1 bytes=%d", stats, file.Size)
	}

	// Ensure file was written.
	path := filepath.Join(tmp, source.DumpFilepath(&ch, &file))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("downloaded file missing: %v", err)
	}
}
