package slackdump

import (
	"context"
	"log"
	"math"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/internal/mocks/mock_os"
	"github.com/rusq/slackdump/v2/internal/network"
)

func Test_validateCache(t *testing.T) {
	type args struct {
		maxAge time.Duration
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(mfi *mock_os.MockFileInfo)
		wantErr  bool
	}{
		{
			"ok",
			args{5 * time.Hour},
			func(mfi *mock_os.MockFileInfo) {
				mfi.EXPECT().IsDir().Return(false)
				mfi.EXPECT().Size().Return(int64(42))
				mfi.EXPECT().ModTime().Return(time.Now())
			},
			false,
		},
		{
			"is dir",
			args{5 * time.Hour},
			func(mfi *mock_os.MockFileInfo) {
				mfi.EXPECT().IsDir().Return(true)
			},
			true,
		},
		{
			"too smol",
			args{5 * time.Hour},
			func(mfi *mock_os.MockFileInfo) {
				mfi.EXPECT().IsDir().Return(false)
				mfi.EXPECT().Size().Return(int64(0))
			},
			true,
		},
		{
			"too old",
			args{5 * time.Hour},
			func(mfi *mock_os.MockFileInfo) {
				mfi.EXPECT().IsDir().Return(false)
				mfi.EXPECT().Size().Return(int64(42))
				mfi.EXPECT().ModTime().Return(time.Now().Add(-10 * time.Hour))
			},
			true,
		},
		{
			"disabled",
			args{0 * time.Hour},
			func(mfi *mock_os.MockFileInfo) {
				mfi.EXPECT().IsDir().Return(false)
				mfi.EXPECT().Size().Return(int64(42))
				mfi.EXPECT().ModTime().Return(time.Now().Add(-1 * time.Nanosecond))
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mfi := mock_os.NewMockFileInfo(ctrl)

			tt.expectFn(mfi)

			if err := validateCache(mfi, tt.args.maxAge); (err != nil) != tt.wantErr {
				t.Errorf("validateFileStats() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_checkCacheFile(t *testing.T) {
	type args struct {
		filename string
		maxAge   time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"empty filename is an error",
			args{"", 1 * time.Hour},
			true,
		},
		// the rest is handled by validateFileStats
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkCacheFile(tt.args.filename, tt.args.maxAge); (err != nil) != tt.wantErr {
				t.Errorf("checkCacheFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
	for _, tt := range tests {
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
	sd, err := New(context.Background(), provider)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}

func ExampleNew_browserAuth() {
	provider, err := auth.NewBrowserAuth()
	if err != nil {
		log.Print(err)
		return
	}
	sd, err := New(context.Background(), provider)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}
