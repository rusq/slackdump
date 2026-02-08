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
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/internal/mocks/mock_auth"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_currentWsp(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name string
		m    *Manager
		args args
		want string
	}{
		{
			"ok",
			&Manager{dir: "test"},
			args{strings.NewReader("foo\n")},
			"foo",
		},
		{
			"empty",
			&Manager{dir: "test"},
			args{strings.NewReader("")},
			defCredsFile,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.readWsp(tt.args.r); got != tt.want {
				t.Errorf("currentWsp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func prepareDir(t *testing.T, dir string) {
	t.Helper()
	fixtures.PrepareDir(t, dir, "dummy", fixtures.WorkspaceFiles...)
}

func testFiles(dir string) []string {
	return fixtures.JoinPath(dir, fixtures.WorkspaceFiles...)
}

func TestManager_listFiles(t *testing.T) {
	tests := []struct {
		name    string
		prepFn  func(t *testing.T, dir string)
		want    func(dir string) []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"ensure that it returns a list of files",
			func(t *testing.T, dir string) {
				prepareDir(t, dir)
			},
			func(dir string) []string {
				return testFiles(dir)
			},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			"empty",
			func(t *testing.T, dir string) {},
			nil,
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return errors.Is(err, ErrNoWorkspaces)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempdir := t.TempDir()
			m := &Manager{
				dir: tempdir,
			}
			if tt.prepFn != nil {
				tt.prepFn(t, tempdir)
			}
			got, err := m.listFiles()
			if !tt.wantErr(t, err, "List()") {
				return
			}
			var want []string
			if tt.want != nil {
				want = tt.want(tempdir)
			}
			sort.Strings(want)
			assert.Equalf(t, want, got, "List()")
		})
	}
}

func TestManager_ExistsErr(t *testing.T) {
	t.Parallel()
	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()

		tempdir := t.TempDir()
		m := &Manager{
			dir: tempdir,
		}
		err := m.ExistsErr("foo")
		assert.ErrorIs(t, err, ErrNoWorkspaces)
	})
	t.Run("workspace exists", func(t *testing.T) {
		t.Parallel()

		tempdir := t.TempDir()
		prepareDir(t, tempdir)
		m := &Manager{
			dir: tempdir,
		}
		err := m.ExistsErr("foo")
		assert.NoError(t, err)
	})
	t.Run("workspace does not exist", func(t *testing.T) {
		t.Parallel()

		tempdir := t.TempDir()
		prepareDir(t, tempdir)
		m := &Manager{
			dir: tempdir,
		}
		err := m.ExistsErr("baz")
		var e *ErrWorkspace
		assert.ErrorAs(t, err, &e)
		assert.Equal(t, e.Message, "no such workspace")
		assert.Equal(t, e.Workspace, "baz")
	})
}

func TestManager_CreateAndSelect(t *testing.T) {
	type fields struct {
		// dir         string
		authOptions []auth.Option
		userFile    string
		channelFile string
	}
	type args struct {
		ctx context.Context
		// prov auth.Provider
	}
	tests := []struct {
		name     string
		fields   fields
		expectFn func(mp *mock_auth.MockProvider)
		args     args
		want     string
		wantErr  bool
	}{
		{
			name: "provider test fails",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(mp *mock_auth.MockProvider) {
				mp.EXPECT().Test(gomock.Any()).Return(nil, assert.AnError)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "url empty fails",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(mp *mock_auth.MockProvider) {
				mp.EXPECT().Test(gomock.Any()).Return(&slack.AuthTestResponse{URL: ""}, nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "url sanitize fails",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(mp *mock_auth.MockProvider) {
				mp.EXPECT().Test(gomock.Any()).Return(&slack.AuthTestResponse{URL: "ftp://lol.example.com"}, nil)
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "success",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(mp *mock_auth.MockProvider) {
				mp.EXPECT().Test(gomock.Any()).Return(fixtures.LoadPtr[slack.AuthTestResponse](string(fixtures.TestAuthTestInfo)), nil)
				mp.EXPECT().Validate().Return(nil)
				mp.EXPECT().SlackToken().Return(fixtures.TestClientToken)
				mp.EXPECT().Cookies().Return([]*http.Cookie{})
			},
			want:    "test",
			wantErr: false,
		},
		{
			name: "save provider fails",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(mp *mock_auth.MockProvider) {
				mp.EXPECT().Test(gomock.Any()).Return(fixtures.LoadPtr[slack.AuthTestResponse](string(fixtures.TestAuthTestInfo)), nil)
				mp.EXPECT().Validate().Return(assert.AnError) // emulate the provider validation error
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			m := &Manager{
				dir:         dir,
				authOptions: tt.fields.authOptions,
				userFile:    tt.fields.userFile,
				channelFile: tt.fields.channelFile,
			}
			ctrl := gomock.NewController(t)
			mp := mock_auth.NewMockProvider(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mp)
			}
			got, err := m.CreateAndSelect(tt.args.ctx, mp)
			if (err != nil) != tt.wantErr {
				t.Errorf("Manager.CreateAndSelect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Manager.CreateAndSelect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_LoadProvider(t *testing.T) {
	t.Run("loads provider", func(t *testing.T) {
		dir := t.TempDir()
		m := &Manager{
			dir: dir,
		}
		prov, err := auth.NewValueAuth(fixtures.TestClientToken, "xoxd-1234567890-1234567890-1234567890-1234567890")
		assert.NoError(t, err)

		m.saveProvider("test.bin", prov)
		got, err := m.LoadProvider("test.bin")
		assert.NoError(t, err)
		assert.Equal(t, prov, got)
	})
	t.Run("encrypted with different machineID", func(t *testing.T) {
		dir := t.TempDir()
		m := &Manager{
			dir: dir,
		}
		prov, err := auth.NewValueAuth(fixtures.TestClientToken, "xoxd-1234567890-1234567890-1234567890-1234567890")
		assert.NoError(t, err)

		m.saveProvider("test.bin", prov)
		m.machineID = "1234567890"
		got, err := m.LoadProvider("test.bin")
		assert.Error(t, err)
		assert.NotEqual(t, prov, got)
	})
	t.Run("encrypted with the same machine ID override", func(t *testing.T) {
		dir := t.TempDir()
		m := &Manager{
			dir:       dir,
			machineID: "1234567890",
		}
		prov, err := auth.NewValueAuth(fixtures.TestClientToken, "xoxd-1234567890-1234567890-1234567890-1234567890")
		assert.NoError(t, err)

		m.saveProvider("test.bin", prov)
		got, err := m.LoadProvider("test.bin")
		assert.NoError(t, err)
		assert.Equal(t, prov, got)
	})
}

func TestManager_createOpener(t *testing.T) {
	type fields struct {
		dir          string
		authOptions  []auth.Option
		userFile     string
		channelFile  string
		machineID    string
		noEncryption bool
	}
	tests := []struct {
		name   string
		fields fields
		want   createOpener
	}{
		{
			"no encryption",
			fields{
				noEncryption: true,
			},
			plainFile{},
		},
		{
			"encrypted",
			fields{
				machineID: "1234567890",
			},
			encryptedFile{machineID: "1234567890"},
		},
		{
			"no machine id",
			fields{},
			encryptedFile{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				dir:          tt.fields.dir,
				authOptions:  tt.fields.authOptions,
				userFile:     tt.fields.userFile,
				channelFile:  tt.fields.channelFile,
				machineID:    tt.fields.machineID,
				noEncryption: tt.fields.noEncryption,
			}
			if got := m.createOpener(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Manager.createOpener() = %v, want %v", got, tt.want)
			}
		})
	}
}
