package downloader

import (
	"context"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"errors"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"golang.org/x/time/rate"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/mocks/mock_downloader"
)

var (
	file1 = slack.File{ID: "f1", Name: "filename1.ext", URLPrivateDownload: "file1_url", Size: 100}
	file2 = slack.File{ID: "f2", Name: "filename2.ext", URLPrivateDownload: "file2_url", Size: 200}
	file3 = slack.File{ID: "f3", Name: "filename3.ext", URLPrivateDownload: "file3_url", Size: 300}
	file4 = slack.File{ID: "f4", Name: "filename4.ext", URLPrivateDownload: "file4_url", Size: 400}
	file5 = slack.File{ID: "f5", Name: "filename5.ext", URLPrivateDownload: "file5_url", Size: 500}
	file9 = slack.File{ID: "f9", Name: "filename9.ext", URLPrivateDownload: "file9_url", Size: 900}
)

// TODO: figure out why this is deprecated.

func TestSession_SaveFileTo(t *testing.T) {
	tmpdir := t.TempDir()

	type fields struct {
		l       *rate.Limiter
		fs      fsadapter.FS
		retries int
		workers int
		nameFn  FilenameFunc
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
				fs:      fsadapter.NewDirectory(tmpdir),
				retries: defRetries,
				workers: defNumWorkers,
				nameFn:  Filename,
			},
			args{
				context.Background(),
				"01",
				&file1,
			},
			func(mc *mock_downloader.MockDownloader) {
				mc.EXPECT().
					GetFile("file1_url", gomock.Any()).
					SetArg(1, *fixtures.FilledFile(t, file1.Size)). // to mock the file size.
					Return(nil)
			},
			int64(file1.Size),
			false,
		},
		{
			"getfile rekt",
			fields{
				l:       rate.NewLimiter(defLimit, 1),
				fs:      fsadapter.NewDirectory(tmpdir),
				retries: defRetries,
				workers: defNumWorkers,
				nameFn:  Filename,
			},
			args{
				context.Background(),
				"02",
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

			sd := &ClientV1{
				v2: &Client{
					sc:      mc,
					fsa:     tt.fields.fs,
					limiter: tt.fields.l,
					retries: tt.fields.retries,
					workers: tt.fields.workers,
				},
				nameFn: tt.fields.nameFn,
			}
			got, err := sd.SaveFile(tt.args.ctx, tt.args.dir, tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.SaveFileTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Session.SaveFileTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_saveFile(t *testing.T) {
	tmpdir := t.TempDir()

	type fields struct {
		l       *rate.Limiter
		fs      fsadapter.FS
		retries int
		workers int
		nameFn  FilenameFunc
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
				fs:      fsadapter.NewDirectory(tmpdir),
				retries: defRetries,
				workers: defNumWorkers,
				nameFn:  Filename,
			},
			args{
				context.Background(),
				"01",
				&file1,
			},
			func(mc *mock_downloader.MockDownloader) {
				mc.EXPECT().
					GetFile("file1_url", gomock.Any()).
					SetArg(1, *fixtures.FilledFile(t, file1.Size)).
					Return(nil)
			},
			int64(file1.Size),
			false,
		},
		{
			"getfile rekt",
			fields{
				l:       rate.NewLimiter(defLimit, 1),
				fs:      fsadapter.NewDirectory(tmpdir),
				retries: defRetries,
				workers: defNumWorkers,
				nameFn:  Filename,
			},
			args{
				context.Background(),
				"02",
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

			sd := &ClientV1{
				v2: &Client{
					sc:      mc,
					fsa:     tt.fields.fs,
					limiter: tt.fields.l,
					retries: tt.fields.retries,
					workers: tt.fields.workers,
				},
				nameFn: tt.fields.nameFn,
			}
			got, err := sd.v2.download(tt.args.ctx, filepath.Join(tt.args.dir, tt.fields.nameFn(tt.args.f)), tt.args.f.URLPrivateDownload)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.saveFileWithLimiter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Session.saveFileWithLimiter() = %v, want %v", got, tt.want)
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
			if got := Filename(tt.args.f); got != tt.want {
				t.Errorf("filename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_newFileDownloader(t *testing.T) {
	tl := rate.NewLimiter(defLimit, 1)
	tmpdir := t.TempDir()

	t.Run("ensure file downloader is running", func(t *testing.T) {
		mc := mock_downloader.NewMockDownloader(gomock.NewController(t))
		sd := ClientV1{
			v2: &Client{
				sc:      mc,
				fsa:     fsadapter.NewDirectory(tmpdir),
				limiter: tl,
				retries: 3,
				workers: 4,
				lg:      slog.Default(),
			},
			nameFn: Filename,
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

		done, err := sd.AsyncDownloader(ctx, ".", filesC)
		require.NoError(t, err)

		<-done
		filename := filepath.Join(tmpdir, Filename(&file9))
		assert.FileExists(t, filename)
	})
}

func TestSession_worker(t *testing.T) {
	tl := rate.NewLimiter(defLimit, 1)
	tmpdir := t.TempDir()

	newClient := func(mc *mock_downloader.MockDownloader) *ClientV1 {
		return &ClientV1{
			v2: &Client{
				sc:      mc,
				fsa:     fsadapter.NewDirectory(tmpdir),
				limiter: tl,
				retries: defRetries,
				workers: defNumWorkers,
				lg:      slog.Default(),
			},
			nameFn: Filename,
		}
	}

	t.Run("sending a single file", func(t *testing.T) {
		mc := mock_downloader.NewMockDownloader(gomock.NewController(t))
		sd := newClient(mc)

		mc.EXPECT().
			GetFile(file1.URLPrivateDownload, gomock.Any()).
			Return(nil).
			Times(1)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		fullpath := filepath.Join(tmpdir, sd.nameFn(&file1))
		t.Log(fullpath)
		reqC := make(chan Request, 1)
		reqC <- Request{Fullpath: sd.nameFn(&file1), URL: file1.URLPrivateDownload}
		close(reqC)

		sd.v2.worker(ctx, reqC)
		assert.FileExists(t, fullpath)
	})
	t.Run("getfile error", func(t *testing.T) {
		mc := mock_downloader.NewMockDownloader(gomock.NewController(t))
		sd := newClient(mc)

		mc.EXPECT().
			GetFile(file1.URLPrivateDownload, gomock.Any()).
			Return(errors.New("rekt")).
			Times(1)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		fullpath := filepath.Join(tmpdir, "01", sd.nameFn(&file1))
		reqC := make(chan Request, 1)
		reqC <- Request{Fullpath: fullpath, URL: file1.URLPrivateDownload}
		close(reqC)

		sd.v2.worker(ctx, reqC)
		_, err := os.Stat(filepath.Join(tmpdir, "01", Filename(&file1)))
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("cancelled context", func(t *testing.T) {
		mc := mock_downloader.NewMockDownloader(gomock.NewController(t))
		sd := newClient(mc)

		reqC := make(chan Request, 1)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		cancel()

		sd.v2.worker(ctx, reqC)
	})
}

func TestClient_startWorkers(t *testing.T) {
	t.Run("check that start actually starts workers", func(t *testing.T) {
		const qSz = 10

		ctrl := gomock.NewController(t)
		dc := mock_downloader.NewMockDownloader(ctrl)
		cl := ClientV1{
			v2: &Client{
				sc:      dc,
				fsa:     fsadapter.NewDirectory(t.TempDir()),
				limiter: rate.NewLimiter(5000, 1),
				workers: defNumWorkers,
				lg:      slog.Default(),
			},
			nameFn: Filename,
		}

		dc.EXPECT().GetFile(gomock.Any(), gomock.Any()).Times(qSz).Return(nil)

		fileQueue := makeFileReqQ(qSz)
		fileChan := slice2chan(fileQueue, defFileBufSz)
		wg := cl.v2.startWorkers(context.Background(), fileChan)

		wg.Wait()
	})
}

// slice2chan takes the slice of []T, create a chan T and sends all elements of
// []T to it.  It closes the channel after all elements are sent.
func slice2chan[T any](input []T, bufSz int) <-chan T {
	output := make(chan T, bufSz)
	go func() {
		defer close(output)
		for _, v := range input {
			output <- v
		}
	}()
	return output
}

func TestClient_Start(t *testing.T) {
	t.Run("make sure structures initialised", func(t *testing.T) {
		c := clientWithMock(t, t.TempDir())

		c.Start(context.Background())
		defer c.Stop()

		assert.True(t, c.v2.started)
		assert.NotNil(t, c.v2.wg)
		assert.NotNil(t, c.v2.requests)
	})
}

func TestClient_Stop(t *testing.T) {
	tmpdir := t.TempDir()
	t.Run("ensure stopped", func(t *testing.T) {
		c := clientWithMock(t, tmpdir)
		c.Start(context.Background())
		assert.True(t, c.v2.started)

		c.Stop()
		assert.False(t, c.v2.started)
		assert.Nil(t, c.v2.requests)
		assert.Nil(t, c.v2.wg)
	})
	t.Run("stop on stopped downloader does nothing", func(t *testing.T) {
		c := clientWithMock(t, tmpdir)
		c.Stop()
		assert.False(t, c.v2.started)
		assert.Nil(t, c.v2.requests)
		assert.Nil(t, c.v2.wg)
	})
}

func clientWithMock(t *testing.T, dir string) *ClientV1 {
	ctrl := gomock.NewController(t)
	dc := mock_downloader.NewMockDownloader(ctrl)
	c := &ClientV1{
		v2: &Client{
			sc:      dc,
			fsa:     fsadapter.NewDirectory(dir),
			limiter: rate.NewLimiter(5000, 1),
			workers: defNumWorkers,
			lg:      slog.Default(),
		},
		nameFn: Filename,
	}
	return c
}

func TestClient_DownloadFile(t *testing.T) {
	dir := t.TempDir()
	t.Run("returns error on stopped downloader", func(t *testing.T) {
		c := clientWithMock(t, dir)
		path, err := c.DownloadFile(dir, slack.File{ID: "xx", Name: "tt"})
		if path != "" {
			t.Errorf("path should be empty")
		}
		if !errors.Is(err, ErrNotStarted) {
			t.Errorf("want err=%s, got=%s", ErrNotStarted, err)
		}
	})
	t.Run("ensure that file is placed on the queue", func(t *testing.T) {
		c := clientWithMock(t, dir)
		c.Start(context.Background())

		c.v2.sc.(*mock_downloader.MockDownloader).EXPECT().
			GetFile(gomock.Any(), gomock.Any()).
			Times(1).
			Return(nil)

		filename, err := c.DownloadFile(dir, file1)
		wantfname := path.Join(dir, Filename(&file1))
		if filename != wantfname {
			t.Errorf("expected filename=%s, got=%s", wantfname, filename)
		}
		if err != nil {
			t.Errorf("error is not expected at this time: %s", err)
		}

		c.Stop()
	})
}
