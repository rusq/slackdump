package slackdump

import (
	"context"
	"log"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/stretchr/testify/assert"
)

func Test_newLimiter(t *testing.T) {
	t.Parallel()
	type args struct {
		t     network.Tier
		burst uint
		boost int
	}
	tests := []struct {
		name      string
		args      args
		wantDelay time.Duration
	}{
		{
			"Tier test",
			args{
				network.Tier3,
				1,
				0,
			},
			time.Duration(math.Round(60.0/float64(network.Tier3)*1000.0)) * time.Millisecond, // 6/5 sec
		},
		{
			"burst 2",
			args{
				network.Tier3,
				2,
				0,
			},
			1 * time.Millisecond,
		},
		{
			"boost 70",
			args{
				network.Tier3,
				1,
				70,
			},
			time.Duration(math.Round(60.0/float64(network.Tier3+70)*1000.0)) * time.Millisecond, // 500 msec
		},
	}
	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := network.NewLimiter(tt.args.t, tt.args.burst, tt.args.boost)

			assert.NoError(t, got.Wait(context.Background())) // prime

			start := time.Now()
			err := got.Wait(context.Background())
			stop := time.Now()

			assert.NoError(t, err)
			assert.WithinDurationf(t, start.Add(tt.wantDelay), stop, 10*time.Millisecond, "delayed for: %s, expected: %s", stop.Sub(start), tt.wantDelay)
		})
	}
}

func ExampleNew_tokenAndCookie() {
	provider, err := auth.NewValueAuth("xoxc-...", "xoxd-...")
	if err != nil {
		log.Print(err)
		return
	}
	sd, err := New(context.Background(), provider, DefOptions)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}

func ExampleNew_cookieFile() {
	provider, err := auth.NewCookieFileAuth("xoxc-...", "cookies.txt")
	if err != nil {
		log.Print(err)
		return
	}
	sd, err := New(context.Background(), provider, DefOptions)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}

func ExampleNew_browserAuth() {
	provider, err := auth.NewBrowserAuth(context.Background())
	if err != nil {
		log.Print(err)
		return
	}
	sd, err := New(context.Background(), provider, DefOptions)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}

func TestSession_openFS(t *testing.T) {
	t.Run("ensure that fs os open and close function added", func(t *testing.T) {
		var sd = new(Session)
		dir := t.TempDir()
		sd.cfg = Config{}
		testFile := filepath.Join(dir, "test.zip")

		assert.NoError(t, sd.openFS(testFile))
		assert.NotNil(t, sd.fs)
		assert.Len(t, sd.atClose, 1)
		assert.NoError(t, sd.Close())
		assert.FileExists(t, testFile)
	})
	t.Run("ensure works with directory", func(t *testing.T) {
		var sd = new(Session)
		dir := t.TempDir()
		sd.cfg = Config{}
		testDir := filepath.Join(dir, "test")

		assert.NoError(t, sd.openFS(testDir))
		assert.NotNil(t, sd.fs)
		assert.Len(t, sd.atClose, 1)

		assert.NoError(t, sd.fs.WriteFile("test.txt", []byte("test"), 0644))

		assert.NoError(t, sd.Close())
		assert.DirExists(t, testDir)
	})
}

func TestSession_openLogger(t *testing.T) {
	t.Run("empty filename should log to stderr", func(t *testing.T) {
		var sd = new(Session)
		sd.cfg = Config{}
		assert.NoError(t, sd.openLogger(""))
		assert.NotNil(t, sd.log)
		assert.Equal(t, sd.log, logger.Default)
		assert.Len(t, sd.atClose, 0) // no close function for stderr
		assert.NoError(t, sd.Close())
	})
	t.Run("non-empty file creates a log file", func(t *testing.T) {
		testLogFile(t, filepath.Join(t.TempDir(), "test.log"), "hello log")
	})
	t.Run("new data is appended to log file if it exists", func(t *testing.T) {
		testFile := filepath.Join(t.TempDir(), "test_again.log")
		testLogFile(t, testFile, "hello log")
		testLogFile(t, testFile, "hello again log")

		data, err := os.ReadFile(testFile)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "hello log")
		assert.Contains(t, string(data), "hello again log")
	})
}

func testLogFile(t *testing.T, testFile string, msg string) {
	var sd = new(Session)
	sd.cfg = Config{}

	assert.NoError(t, sd.openLogger(testFile))
	assert.NotNil(t, sd.log)
	assert.Len(t, sd.atClose, 1)

	sd.log.Print(msg)

	assert.NoError(t, sd.Close())
	assert.FileExists(t, testFile)

	data, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Contains(t, string(data), msg)
}
