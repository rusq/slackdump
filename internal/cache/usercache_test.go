package cache

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rusq/encio"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/rusq/slackdump/v2/internal/mocks/mock_os"
	"github.com/rusq/slackdump/v2/types"
)

const testSuffix = "UNIT"

var testUsers = types.Users(fixtures.TestUsers)

func TestSaveUserCache(t *testing.T) {
	// test saving file works
	dir := t.TempDir()
	testfile := "test.json"

	assert.NoError(t, saveUsers(dir, testfile, testSuffix, testUsers))

	reopenedF, err := encio.Open(makeCacheFilename(dir, testfile, testSuffix))
	if err != nil {
		t.Fatal(err)
	}
	defer reopenedF.Close()
	uu, err := read[slack.User](reopenedF)
	assert.NoError(t, err)
	assert.Equal(t, testUsers, types.Users(uu))
}

func TestLoadUserCache(t *testing.T) {
	dir := t.TempDir()
	type args struct {
		filename string
		maxAge   time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    types.Users
		wantErr bool
	}{
		{
			"loads the cache ok",
			args{gimmeTempFileWithUsers(t, dir), 5 * time.Hour},
			testUsers,
			false,
		},
		{
			"no data",
			args{gimmeTempFile(t, dir), 5 * time.Hour},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadUsers("", tt.args.filename, testSuffix, tt.args.maxAge)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.loadUserCache() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Session.loadUserCache() = %v, want %v", got, tt.want)
			}
		})
	}
}

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

func gimmeTempFile(t *testing.T, dir string) string {
	f, err := os.CreateTemp(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Errorf("error closing test file: %s", err)
	}
	return f.Name()
}

func gimmeTempFileWithUsers(t *testing.T, dir string) string {
	f := gimmeTempFile(t, dir)
	if err := saveUsers("", f, testSuffix, testUsers); err != nil {
		t.Fatal(err)
	}
	return f
}

func FuzzFilenameSplit(f *testing.F) {
	testInput := []string{
		"users.json",
		"channels.json",
	}
	for _, ti := range testInput {
		f.Add(ti)
	}
	f.Fuzz(func(t *testing.T, input string) {
		split := filenameSplit(input)
		joined := filenameJoin(split)
		assert.Equal(t, input, joined)
	})
}
