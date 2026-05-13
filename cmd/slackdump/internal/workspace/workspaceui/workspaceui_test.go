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

package workspaceui

import (
	"path/filepath"
	"testing"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/cfgui"
)

type errorer interface {
	Err() error
}

func TestCacheOptionsCacheDirectory(t *testing.T) {
	oldCacheDir := cfg.LocalCacheDir
	t.Cleanup(func() { cfg.LocalCacheDir = oldCacheDir })

	t.Run("accepts existing directory", func(t *testing.T) {
		cfg.LocalCacheDir = t.TempDir()

		param := cacheDirectoryParam(t)

		if param.Name != "Cache Directory" {
			t.Fatalf("Name = %q, want %q", param.Name, "Cache Directory")
		}
		if param.Value != cfg.LocalCacheDir {
			t.Fatalf("Value = %q, want %q", param.Value, cfg.LocalCacheDir)
		}
		if !param.Inline {
			t.Fatal("Inline = false, want true")
		}
		if param.Updater == nil {
			t.Fatal("Updater = nil, want directory validator")
		}
		updater, ok := param.Updater.(errorer)
		if !ok {
			t.Fatalf("Updater = %T, want Err() error", param.Updater)
		}
		if err := updater.Err(); err != nil {
			t.Fatalf("Updater.Err() = %v, want nil", err)
		}
	})

	t.Run("rejects missing directory", func(t *testing.T) {
		cfg.LocalCacheDir = filepath.Join(t.TempDir(), "missing")

		param := cacheDirectoryParam(t)
		updater, ok := param.Updater.(errorer)
		if !ok {
			t.Fatalf("Updater = %T, want Err() error", param.Updater)
		}
		if err := updater.Err(); err == nil {
			t.Fatal("Updater.Err() = nil, want missing directory error")
		}
	})
}

func cacheDirectoryParam(t *testing.T) cfgui.Parameter {
	t.Helper()

	conf := cacheOptions()
	if len(conf) != 1 {
		t.Fatalf("len(cacheOptions()) = %d, want 1", len(conf))
	}
	for _, param := range conf[0].Params {
		if param.Name == "Cache Directory" {
			return param
		}
	}
	t.Fatal("Cache Directory parameter not found")
	return cfgui.Parameter{}
}
