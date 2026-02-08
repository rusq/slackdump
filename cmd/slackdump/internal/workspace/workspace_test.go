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
package workspace

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/workspace/workspaceui"
	"github.com/rusq/slackdump/v4/internal/cache"
	fx "github.com/rusq/slackdump/v4/internal/fixtures"
)

func Test_argsWorkspace(t *testing.T) {
	type args struct {
		args       []string
		defaultWsp string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"empty",
			args{[]string{}, ""},
			"",
		},
		{
			"default is set, no workspace in args",
			args{[]string{}, "default"},
			"default",
		},
		{
			"default overrides args args",
			args{[]string{"arg"}, "default"},
			"default",
		},
		{
			"returns must be lowercase",
			args{[]string{"UPPERCASE"}, "DEFAULT"},
			"default",
		},
		{
			"returns must be lowercase",
			args{[]string{"UPPERCASE"}, ""},
			"uppercase",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := argsWorkspace(tt.args.args, tt.args.defaultWsp); got != tt.want {
				t.Errorf("argsWorkspace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCurrent(t *testing.T) {
	// fixtures

	empty := t.TempDir()

	// case1 has files, but no file pointing to the current workspace
	case1 := t.TempDir()
	fx.PrepareDir(t, case1, "dummy", fx.WorkspaceFiles...)

	// case2 has files, and a pointer to the current workspace.
	case2 := t.TempDir()
	fx.PrepareDir(t, case2, "dummy", fx.WorkspaceFiles...)
	os.WriteFile(filepath.Join(case2, "workspace.txt"), []byte(fx.StripExt(fx.WorkspaceFiles[0])+"\n"), 0o644)

	// case3 has a file, which is specified as a directory to the function
	// so that manager fails to initialise.
	case3 := t.TempDir()
	os.WriteFile(filepath.Join(case3, "cache_dir"), []byte(""), 0o644)

	// case4 workspace pointer points to non-existing file.
	case4 := t.TempDir()
	fx.PrepareDir(t, case4, "dummy", fx.WorkspaceFiles...)
	os.WriteFile(filepath.Join(case4, "workspace.txt"), []byte("doesnotexist\n"), 0o644)

	// tests
	type args struct {
		cacheDir string
		override string
	}
	tests := []struct {
		name    string
		args    args
		wantWsp string
		wantErr bool
	}{
		{
			"empty,no override",
			args{empty, ""},
			"default",
			false,
		},
		{
			"override, does not exist",
			args{empty, "override"},
			"",
			true,
		},
		{
			"case1, no override",
			args{case1, ""},
			"default",
			false,
		},
		{
			"case2, no override",
			args{case2, ""},
			fx.StripExt(fx.WorkspaceFiles[0]),
			false,
		},
		{
			"case2, override",
			args{case2, fx.StripExt(fx.WorkspaceFiles[1])},
			fx.StripExt(fx.WorkspaceFiles[1]),
			false,
		},
		{
			"case2, override, does not exist",
			args{case2, "doesnotexist"},
			"",
			true,
		},
		{
			"invalid directory",
			args{filepath.Join(case3, "cache_dir"), ""},
			"",
			true,
		},
		{
			"case4, no override, returns default",
			args{case4, ""},
			"default",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWsp, err := Current(tt.args.cacheDir, tt.args.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("Current() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotWsp != tt.wantWsp {
				t.Errorf("Current() = %v, want %v", gotWsp, tt.wantWsp)
			}
		})
	}
}

type recorder struct {
	authCurrentCalledTimes int
	authCurrentRetProv     auth.Provider
	authCurrentRetErr      error

	showUICalledTimes int
	showUIRetErr      error
}

func (r *recorder) AuthCurrent(ctx context.Context, cacheDir string, overrideWsp string, usePlaywright bool) (auth.Provider, error) {
	r.authCurrentCalledTimes++
	return r.authCurrentRetProv, r.authCurrentRetErr
}

func (r *recorder) ShowUI(ctx context.Context, opts ...workspaceui.UIOption) error {
	r.showUICalledTimes++
	return r.showUIRetErr
}

func TestCurrentOrNewProviderCtx(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name                       string
		args                       args
		rec                        *recorder
		want                       context.Context
		wantErr                    bool
		wantAuthCurrentCalledTimes int
		wantShowUICalledTimes      int
	}{
		{
			"authCurrent fails",
			args{t.Context()},
			&recorder{
				authCurrentRetErr: assert.AnError,
			},
			t.Context(),
			true,
			1,
			0,
		},
		{
			"authCurrent doesn't find workspace",
			args{t.Context()},
			&recorder{
				authCurrentRetErr: cache.ErrNoWorkspaces,
			},
			t.Context(),
			true,
			2, // attempts to call authCurrent twice
			1, // after showing the UI
		},
		{
			"authCurrent succeeds",
			args{t.Context()},
			&recorder{
				authCurrentRetProv: auth.ValueAuth{},
			},
			auth.WithContext(t.Context(), auth.ValueAuth{}),
			false,
			1,
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := tt.rec
			authCurrent = rec.AuthCurrent
			showUI = rec.ShowUI

			got, err := CurrentOrNewProviderCtx(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("CurrentOrNewProviderCtx() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CurrentOrNewProviderCtx() = %v, want %v", got, tt.want)
			}
			assert.Equal(t, rec.authCurrentCalledTimes, tt.wantAuthCurrentCalledTimes)
			assert.Equal(t, rec.showUICalledTimes, tt.wantShowUICalledTimes)
		})
	}
}
