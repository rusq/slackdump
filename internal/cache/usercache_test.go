// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package cache

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/internal/mocks/mock_os"
	"github.com/rusq/slackdump/v4/types"
)

const testSuffix = "UNIT"

var testUsers = types.Users(fixtures.TestUsers)

func TestSaveUserCache(t *testing.T) {
	// test saving file works
	dir := t.TempDir()
	testfile := "test.json"

	var m Manager
	assert.NoError(t, m.saveUsers(dir, testfile, testSuffix, testUsers))

	reopenedF, err := m.createOpener().Open(makeCacheFilename(dir, testfile, testSuffix))
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
			var m Manager
			got, err := m.loadUsers("", tt.args.filename, testSuffix, tt.args.maxAge)
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
	var m Manager
	if err := m.saveUsers("", f, testSuffix, testUsers); err != nil {
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
