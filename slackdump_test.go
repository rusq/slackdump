package slackdump

import (
	"context"
	"log"
	"math"
	"os"
	"testing"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/internal/network"
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
	fsa := openTempFS()
	defer fsa.Close()

	sd, err := New(context.Background(), provider)
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
	fsa := openTempFS()
	defer fsa.Close()

	sd, err := New(context.Background(), provider)
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
	fsa := openTempFS()
	defer fsa.Close()
	sd, err := New(context.Background(), provider)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}

func openTempFS() fsadapter.FSCloser {
	dir, err := os.MkdirTemp("", "slackdump")
	if err != nil {
		panic(err)
	}
	fs, err := fsadapter.New(dir)
	if err != nil {
		panic(err)
	}
	return fs
}
