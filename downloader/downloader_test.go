package downloader

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/fixtures"
)

func init() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

func Test_fltSeen(t *testing.T) {
	t.Run("ensure that we don't get dup files", func(t *testing.T) {
		source := []Request{
			{Fullpath: "x/file1", URL: "url1"},
			{Fullpath: "x/file2", URL: "url2"},
			{Fullpath: "a/file2", URL: "url2"}, // different path
			{Fullpath: "x/file3", URL: "url3"},
			{Fullpath: "x/file4", URL: "url4"},
			{Fullpath: "x/file5", URL: "url5"},
			{Fullpath: "y/file5", URL: "url5"},
			{Fullpath: "x/file1", URL: "url2"}, // different url same path
			// duplicates
			{Fullpath: "x/file1", URL: "url1"},
			{Fullpath: "x/file2", URL: "url2"},
			{Fullpath: "a/file2", URL: "url2"},
		}
		want := []Request{
			{Fullpath: "x/file1", URL: "url1"},
			{Fullpath: "x/file2", URL: "url2"},
			{Fullpath: "a/file2", URL: "url2"},
			{Fullpath: "x/file3", URL: "url3"},
			{Fullpath: "x/file4", URL: "url4"},
			{Fullpath: "x/file5", URL: "url5"},
			{Fullpath: "y/file5", URL: "url5"},
			{Fullpath: "x/file1", URL: "url2"},
		}

		filesC := make(chan Request)
		go func() {
			defer close(filesC)
			for _, f := range source {
				filesC <- f
			}
		}()

		dlqC := fltSeen(filesC)

		var got []Request
		for f := range dlqC {
			got = append(got, f)
		}
		assert.Equal(t, want, got)
	})
}

var benchInput = makeFileReqQ(100_000)

func BenchmarkFltSeen(b *testing.B) {
	inputC := make(chan Request)
	go func() {
		defer close(inputC)
		for _, req := range benchInput {
			inputC <- req
		}
	}()
	outputC := fltSeen(inputC)

	for n := 0; n < b.N; n++ {
		for out := range outputC {
			_ = out
		}
	}
}

func makeFileReqQ(numReq int) []Request {
	reqQ := make([]Request, numReq)
	for i := 0; i < numReq; i++ {
		reqQ[i] = Request{Fullpath: fixtures.RandString(8), URL: fixtures.RandString(16)}
	}
	return reqQ
}

func TestClient_Stop(t *testing.T) {
	t.Run("already stopped", func(t *testing.T) {
		c := &Client{
			requests: make(chan Request),
			wg:       new(sync.WaitGroup),
			options:  options{lg: slog.Default()},
		}
		c.started.Store(true)
		c.Stop()
		assert.False(t, c.started.Load(), "expected started to be false")
		// shouldn't panic because the channel is closed
		assert.NotPanics(t, c.Stop)
		assert.False(t, c.started.Load(), "expected started to be false")
	})
}

func TestClient_Download(t *testing.T) {
	t.Run("not started", func(t *testing.T) {
		c := &Client{
			requests: make(chan Request),

			wg:      new(sync.WaitGroup),
			options: options{lg: slog.Default()},
		}
		err := c.Download("x/file", "http://example.com")
		assert.Error(t, err, "expected error")
	})
	t.Run("started", func(t *testing.T) {
		requests := make(chan Request, 1)
		c := &Client{
			requests: requests,
			wg:       new(sync.WaitGroup),
			options:  options{lg: slog.Default()},
		}
		c.started.Store(true)
		err := c.Download("x/file", "http://example.com")
		assert.NoError(t, err, "expected no error")
		tm := time.NewTicker(1 * time.Second)
		select {
		case <-tm.C:
			t.Fatal("expected request to be sent")
		case r := <-requests:
			assert.Equal(t, Request{Fullpath: "x/file", URL: "http://example.com"}, r, "expected request to be sent")
		}
	})
}

func TestClient_startWorkers(t *testing.T) {
	t.Parallel()
	t.Run("starts workers", func(t *testing.T) {
		t.Parallel()
		c := &Client{
			requests: make(chan Request),
			wg:       new(sync.WaitGroup),
			options:  options{lg: slog.Default(), workers: 3},
		}
		defer close(c.requests)
		c.startWorkers(context.Background())
		assert.Equal(t, 3, c.options.workers)
		assert.NotNil(t, c.wg)
		assert.True(t, c.started.Load())
	})
	t.Run("no workers specified", func(t *testing.T) {
		t.Parallel()
		c := &Client{
			requests: make(chan Request),
			wg:       new(sync.WaitGroup),
			options:  options{lg: slog.Default()},
		}
		defer close(c.requests)
		c.startWorkers(context.Background())
		assert.Equal(t, defNumWorkers, c.options.workers)
		assert.NotNil(t, c.wg)
		assert.True(t, c.started.Load())
	})
}
