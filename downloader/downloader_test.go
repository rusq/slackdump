package downloader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/rusq/slackdump/v2/internal/mock_downloader"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

var (
	file1 = slack.File{ID: "f1", Name: "filename1.ext", URLPrivateDownload: "file1_url", Size: 100}
	file2 = slack.File{ID: "f2", Name: "filename2.ext", URLPrivateDownload: "file2_url", Size: 200}
	file3 = slack.File{ID: "f3", Name: "filename3.ext", URLPrivateDownload: "file3_url", Size: 300}
	file4 = slack.File{ID: "f4", Name: "filename4.ext", URLPrivateDownload: "file4_url", Size: 400}
	file5 = slack.File{ID: "f5", Name: "filename5.ext", URLPrivateDownload: "file5_url", Size: 500}
	file6 = slack.File{ID: "f6", Name: "filename6.ext", URLPrivateDownload: "file6_url", Size: 600}
	file7 = slack.File{ID: "f7", Name: "filename7.ext", URLPrivateDownload: "file7_url", Size: 700}
	file8 = slack.File{ID: "f8", Name: "filename8.ext", URLPrivateDownload: "file8_url", Size: 800}
	file9 = slack.File{ID: "f9", Name: "filename9.ext", URLPrivateDownload: "file9_url", Size: 900}
)

func TestSlackDumper_SaveFileTo(t *testing.T) {
	tmpdir := t.TempDir()

	type fields struct {
		l       *rate.Limiter
		retries int
		workers int
	}
	type args struct {
		ctx context.Context
		dir string
		f   *slack.File
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mc *mock_downloader.MockDownloader)
		want     int64
		wantErr  bool
	}{
		{
			"ok",
			fields{
				l:       rate.NewLimiter(defLimit, 1),
				retries: defRetries,
				workers: defNumWorkers},
			args{
				context.Background(),
				tmpdir,
				&file1,
			},
			func(mc *mock_downloader.MockDownloader) {
				mc.EXPECT().
					GetFile("file1_url", gomock.Any()).
					Return(nil)
			},
			int64(file1.Size),
			false,
		},
		{
			"getfile rekt",
			fields{
				l:       rate.NewLimiter(defLimit, 1),
				retries: defRetries,
				workers: defNumWorkers},
			args{
				context.Background(),
				tmpdir,
				&file2,
			},
			func(mc *mock_downloader.MockDownloader) {
				mc.EXPECT().
					GetFile("file2_url", gomock.Any()).
					Return(errors.New("rekt"))
			},
			int64(0),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := mock_downloader.NewMockDownloader(ctrl)

			tt.expectFn(mc)

			sd := &Client{
				client:  mc,
				limiter: tt.fields.l,
				retries: tt.fields.retries,
				workers: tt.fields.workers,
			}
			got, err := sd.SaveFileTo(tt.args.ctx, tt.args.dir, tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.SaveFileTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SlackDumper.SaveFileTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackDumper_saveFile(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	type fields struct {
		l       *rate.Limiter
		retries int
		workers int
	}
	type args struct {
		ctx context.Context
		dir string
		f   *slack.File
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mc *mock_downloader.MockDownloader)
		want     int64
		wantErr  bool
	}{
		{
			"ok",
			fields{
				l:       rate.NewLimiter(defLimit, 1),
				retries: defRetries,
				workers: defNumWorkers,
			},
			args{
				context.Background(),
				tmpdir,
				&file1,
			},
			func(mc *mock_downloader.MockDownloader) {
				mc.EXPECT().
					GetFile("file1_url", gomock.Any()).
					Return(nil)
			},
			int64(file1.Size),
			false,
		},
		{
			"getfile rekt",
			fields{
				l:       rate.NewLimiter(defLimit, 1),
				retries: defRetries,
				workers: defNumWorkers,
			},
			args{
				context.Background(),
				tmpdir,
				&file2,
			},
			func(mc *mock_downloader.MockDownloader) {
				mc.EXPECT().
					GetFile("file2_url", gomock.Any()).
					Return(errors.New("rekt"))
			},
			int64(0),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := mock_downloader.NewMockDownloader(ctrl)

			tt.expectFn(mc)

			sd := &Client{
				client:  mc,
				limiter: tt.fields.l,
				retries: tt.fields.retries,
				workers: tt.fields.workers,
			}
			got, err := sd.saveFile(tt.args.ctx, tt.args.dir, tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.saveFileWithLimiter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SlackDumper.saveFileWithLimiter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_filename(t *testing.T) {
	type args struct {
		f *slack.File
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"file1", args{&file1}, "f1-filename1.ext"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filename(tt.args.f); got != tt.want {
				t.Errorf("filename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackDumper_newFileDownloader(t *testing.T) {
	t.Parallel()
	tl := rate.NewLimiter(defLimit, 1)
	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	t.Run("ensure file downloader is running", func(t *testing.T) {
		mc := mock_downloader.NewMockDownloader(gomock.NewController(t))
		sd := Client{
			client:  mc,
			limiter: tl,
			retries: 3,
			workers: 4,
		}

		mc.EXPECT().
			GetFile(file9.URLPrivateDownload, gomock.Any()).
			Return(nil).
			Times(1)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Second)
		defer cancel()
		filesC := make(chan *slack.File, 1)
		filesC <- &file9
		close(filesC)

		done, err := sd.AsyncDownloader(ctx, tmpdir, filesC)
		require.NoError(t, err)

		<-done
		filename := filepath.Join(tmpdir, filename(&file9))
		assert.FileExists(t, filename)

	})
}

func TestSlackDumper_worker(t *testing.T) {
	t.Parallel()
	tl := rate.NewLimiter(defLimit, 1)
	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	t.Run("sending a single file", func(t *testing.T) {
		mc := mock_downloader.NewMockDownloader(gomock.NewController(t))
		sd := Client{
			client:  mc,
			limiter: tl,
			retries: defRetries,
			workers: defNumWorkers,
		}

		mc.EXPECT().
			GetFile(file1.URLPrivateDownload, gomock.Any()).
			Return(nil).
			Times(1)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		filesC := make(chan *slack.File, 1)
		filesC <- &file1
		close(filesC)

		sd.worker(ctx, tmpdir, filesC)
		assert.FileExists(t, filepath.Join(tmpdir, filename(&file1)))
	})
	t.Run("getfile error", func(t *testing.T) {
		mc := mock_downloader.NewMockDownloader(gomock.NewController(t))
		sd := Client{
			client:  mc,
			limiter: tl,
			retries: defRetries,
			workers: defNumWorkers,
		}

		mc.EXPECT().
			GetFile(file1.URLPrivateDownload, gomock.Any()).
			Return(errors.New("rekt")).
			Times(1)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		filesC := make(chan *slack.File, 1)
		filesC <- &file1
		close(filesC)

		sd.worker(ctx, tmpdir, filesC)
		_, err := os.Stat(filepath.Join(tmpdir, filename(&file1)))
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("cancelled context", func(t *testing.T) {
		mc := mock_downloader.NewMockDownloader(gomock.NewController(t))
		sd := Client{
			client:  mc,
			limiter: tl,
			retries: defRetries,
			workers: defNumWorkers,
		}

		filesC := make(chan *slack.File, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		cancel()

		sd.worker(ctx, tmpdir, filesC)
	})
}

func Test_seenFilter(t *testing.T) {
	t.Run("ensure that we don't get dup files", func(t *testing.T) {
		source := []slack.File{file1, file2, file2, file3, file3, file3, file4, file5}
		want := []slack.File{file1, file2, file3, file4, file5}

		filesC := make(chan *slack.File)
		go func() {
			defer close(filesC)
			for _, f := range source {
				file := f // copy
				filesC <- &file
			}
		}()
		dlqC := seenFilter(filesC)

		var got []slack.File
		for f := range dlqC {
			got = append(got, *f)
		}
		assert.Equal(t, want, got)
	})
}
