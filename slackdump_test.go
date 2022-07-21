package slackdump

import (
	"context"
	"log"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/rusq/slackdump/v2/internal/mocks/mock_os"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
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
	provider, err := auth.NewBrowserAuth(context.Background())
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

func TestSession_Me(t *testing.T) {
	type fields struct {
		client    clienter
		fs        fsadapter.FS
		wspInfo   *slack.AuthTestResponse
		Users     types.Users
		UserIndex structures.UserIndex
		options   Options
		cacheDir  string
	}
	tests := []struct {
		name    string
		fields  fields
		want    slack.User
		wantErr bool
	}{
		{
			"all ok",
			fields{
				wspInfo:   &slack.AuthTestResponse{UserID: "DELD"},
				UserIndex: structures.NewUserIndex(fixtures.TestUsers),
			},
			fixtures.TestUsers[1],
			false,
		},
		{
			"no users - error",
			fields{Users: nil},
			slack.User{},
			true,
		},
		{
			"not enough users - error",
			fields{UserIndex: structures.UserIndex{}},
			slack.User{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &Session{
				client:    tt.fields.client,
				fs:        tt.fields.fs,
				wspInfo:   tt.fields.wspInfo,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
				cacheDir:  tt.fields.cacheDir,
			}
			got, err := sd.Me()
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.Me() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Session.Me() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_l(t *testing.T) {
	testLg := dlog.New(os.Stderr, "TEST", log.LstdFlags, false)
	type fields struct {
		client    clienter
		wspInfo   *slack.AuthTestResponse
		fs        fsadapter.FS
		Users     types.Users
		UserIndex structures.UserIndex
		options   Options
		cacheDir  string
	}
	tests := []struct {
		name   string
		fields fields
		want   logger.Interface
	}{
		{
			"empty returns the default logger",
			fields{
				options: Options{},
			},
			logger.Default,
		},
		{
			"if logger is set, it returns the custom logger",
			fields{
				options: Options{Logger: testLg},
			},
			logger.Interface(testLg),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &Session{
				client:    tt.fields.client,
				wspInfo:   tt.fields.wspInfo,
				fs:        tt.fields.fs,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
				cacheDir:  tt.fields.cacheDir,
			}
			if got := sd.l(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Session.l() = %v, want %v", got, tt.want)
			}
		})
	}
}
