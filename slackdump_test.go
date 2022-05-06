package slackdump

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rusq/slackdump/v2/internal/mocks/mock_os"
	"github.com/stretchr/testify/assert"
)

func Test_maxStringLength(t *testing.T) {
	type args struct {
		strings []string
	}
	tests := []struct {
		name       string
		args       args
		wantMaxlen int
	}{
		{"ascii", args{[]string{"123", "abc", "defg"}}, 4},
		{"unicode", args{[]string{"сообщение1", "проверка", "тест"}}, 10},
		{"empty", args{[]string{}}, 0},
		{"several empty", args{[]string{"", "", "", ""}}, 0},
		{"several empty one full", args{[]string{"", "", "1", ""}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotMaxlen := maxStringLength(tt.args.strings); gotMaxlen != tt.wantMaxlen {
				t.Errorf("maxStringLength() = %v, want %v", gotMaxlen, tt.wantMaxlen)
			}
		})
	}
}

func Test_validateFileStats(t *testing.T) {
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

			if err := validateFileStats(mfi, tt.args.maxAge); (err != nil) != tt.wantErr {
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
		t     tier
		burst uint
		boost int
	}
	tests := []struct {
		name      string
		args      args
		wantDelay time.Duration
	}{
		{
			"tier test",
			args{
				tier3,
				1,
				0,
			},
			time.Duration(math.Round(60.0/float64(tier3)*1000.0)) * time.Millisecond, // 6/5 sec
		},
		{
			"burst 2",
			args{
				tier3,
				2,
				0,
			},
			1 * time.Millisecond,
		},
		{
			"boost 70",
			args{
				tier3,
				1,
				70,
			},
			time.Duration(math.Round(60.0/float64(tier3+70)*1000.0)) * time.Millisecond, // 500 msec
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := newLimiter(tt.args.t, tt.args.burst, tt.args.boost)

			assert.NoError(t, got.Wait(context.Background())) // prime

			start := time.Now()
			err := got.Wait(context.Background())
			stop := time.Now()

			assert.NoError(t, err)
			assert.WithinDurationf(t, start.Add(tt.wantDelay), stop, 10*time.Millisecond, "delayed for: %s, expected: %s", stop.Sub(start), tt.wantDelay)
		})
	}
}

func Test_isExistingFile(t *testing.T) {
	testfile := filepath.Join(t.TempDir(), "cookies.txt")
	if err := os.WriteFile(testfile, []byte("blah"), 0600); err != nil {
		t.Fatal(err)
	}

	type args struct {
		cookie string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"not a file", args{"$blah"}, false},
		{"file", args{testfile}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExistingFile(tt.args.cookie); got != tt.want {
				t.Errorf("isExistingFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
