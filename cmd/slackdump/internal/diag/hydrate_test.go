package diag

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rusq/slack"
	gomock "go.uber.org/mock/gomock"

	"github.com/rusq/fsadapter/mocks/mock_fsadapter"

	"github.com/rusq/slackdump/v3/mocks/mock_downloader"
)

func init() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

func Test_httpget_GetFileContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer srv.Close()

	errsrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error", http.StatusInternalServerError)
	}))
	defer errsrv.Close()

	type args struct {
		ctx         context.Context
		downloadURL string
	}
	tests := []struct {
		name    string
		h       httpget
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name: "invalid URL",
			h:    httpget{},
			args: args{
				ctx:         context.Background(),
				downloadURL: "invalidURL",
			},
			wantW:   "",
			wantErr: true,
		},
		{
			name: "missing token in the URL",
			h:    httpget{},
			args: args{
				ctx:         context.Background(),
				downloadURL: srv.URL,
			},
			wantW:   "",
			wantErr: true,
		},
		{
			name: "all ok",
			h:    httpget{},
			args: args{
				ctx:         context.Background(),
				downloadURL: srv.URL + "?t=token",
			},
			wantW:   "Hello, client\n",
			wantErr: false,
		},
		{
			name: "error",
			h:    httpget{},
			args: args{
				ctx:         context.Background(),
				downloadURL: errsrv.URL + "?t=token",
			},
			wantW:   "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := httpget{}
			w := &bytes.Buffer{}
			if err := h.GetFileContext(tt.args.ctx, tt.args.downloadURL, w); (err != nil) != tt.wantErr {
				t.Errorf("httpget.GetFileContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("httpget.GetFileContext() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

var (
	TestFile1 = slack.File{ID: "1", Name: "file1", URLPrivateDownload: "testURL1"}
	TestFile2 = slack.File{ID: "2", Name: "file2", URLPrivateDownload: "testURL2"}
	TestFile3 = slack.File{ID: "3", Name: "file3", URLPrivateDownload: "testURL3"}
	TestFile4 = slack.File{ID: "4", Name: "file4", URLPrivateDownload: "testURL4"}

	TestMsgWFile1 = slack.Message{
		Msg: slack.Msg{
			Timestamp: "1",
			Files:     []slack.File{TestFile1, TestFile2},
		},
	}
	TestThreadStartWFile = slack.Message{
		Msg: slack.Msg{
			Timestamp:       "2",
			ThreadTimestamp: "2",
			Files:           []slack.File{TestFile3},
		},
	}
	TestThreadMsgWFile = slack.Message{
		Msg: slack.Msg{
			ThreadTimestamp: "2",
			Files:           []slack.File{TestFile4},
		},
	}

	TestChannels = []slack.Channel{
		{GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{ID: "C01"},
			Name:         "channel1",
		}},
	}
	TestMessages       = []slack.Message{TestMsgWFile1, TestThreadStartWFile}
	TestThreadMessages = []slack.Message{TestThreadMsgWFile}
)

type fakewritecloser struct{}

func (f *fakewritecloser) Write(p []byte) (n int, err error) {
	return 0, nil
}

func (f *fakewritecloser) Close() error {
	return nil
}

func Test_downloadFiles(t *testing.T) {
	tests := []struct {
		name     string
		expectFn func(m *Mocksourcer, fs *mock_fsadapter.MockFSCloser, d *mock_downloader.MockGetFiler)
		wantErr  bool
	}{
		{
			"single message w 2 files",
			func(m *Mocksourcer, fs *mock_fsadapter.MockFSCloser, d *mock_downloader.MockGetFiler) {
				m.EXPECT().Channels(gomock.Any()).Return(TestChannels, nil)
				m.EXPECT().AllMessages(gomock.Any(), "C01").Return([]slack.Message{TestMsgWFile1}, nil)

				fs.EXPECT().Create(filepath.Join("__uploads", "1", "file1")).Return(&fakewritecloser{}, nil)
				fs.EXPECT().Create(filepath.Join("__uploads", "2", "file2")).Return(&fakewritecloser{}, nil)

				d.EXPECT().GetFileContext(gomock.Any(), "testURL1", gomock.Any()).Return(nil)
				d.EXPECT().GetFileContext(gomock.Any(), "testURL2", gomock.Any()).Return(nil)
			},
			false,
		},
		{
			"all ok",
			func(m *Mocksourcer, fs *mock_fsadapter.MockFSCloser, d *mock_downloader.MockGetFiler) {
				m.EXPECT().Channels(gomock.Any()).Return(TestChannels, nil)
				m.EXPECT().AllMessages(gomock.Any(), "C01").Return(TestMessages, nil)
				m.EXPECT().AllThreadMessages(gomock.Any(), "C01", "2").Return(TestThreadMessages, nil)

				fs.EXPECT().Create(filepath.Join("__uploads", "1", "file1")).Return(&fakewritecloser{}, nil)
				fs.EXPECT().Create(filepath.Join("__uploads", "2", "file2")).Return(&fakewritecloser{}, nil)
				fs.EXPECT().Create(filepath.Join("__uploads", "3", "file3")).Return(&fakewritecloser{}, nil)
				fs.EXPECT().Create(filepath.Join("__uploads", "4", "file4")).Return(&fakewritecloser{}, nil)

				d.EXPECT().GetFileContext(gomock.Any(), "testURL1", gomock.Any()).Return(nil)
				d.EXPECT().GetFileContext(gomock.Any(), "testURL2", gomock.Any()).Return(nil)
				d.EXPECT().GetFileContext(gomock.Any(), "testURL3", gomock.Any()).Return(nil)
				d.EXPECT().GetFileContext(gomock.Any(), "testURL4", gomock.Any()).Return(nil)
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := NewMocksourcer(ctrl)
			fs := mock_fsadapter.NewMockFSCloser(ctrl)
			d := mock_downloader.NewMockGetFiler(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(ms, fs, d)
			}
			if err := downloadFiles(context.Background(), d, fs, ms); (err != nil) != tt.wantErr {
				t.Errorf("downloadFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
