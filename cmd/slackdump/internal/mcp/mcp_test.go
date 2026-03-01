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

package mcp

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFS is a minimal in-memory FS used across tests to avoid depending on
// the real embedded assets.
var testFS = fstest.MapFS{
	"opencode.jsonc":                  {Data: []byte(`{"mcp":{}}`)},
	".opencode/skills/skill/SKILL.md": {Data: []byte("# skill\n")},
}

// ─── initNewProject ───────────────────────────────────────────────────────────

func Test_initNewProject_CreatesTargetDir(t *testing.T) {
	tgt := filepath.Join(t.TempDir(), "new-project")
	// tgt must not exist yet
	require.NoDirExists(t, tgt)

	err := initNewProject(tgt, testFS)
	require.NoError(t, err)

	assert.DirExists(t, tgt)
}

func Test_initNewProject_CopiesFiles(t *testing.T) {
	tgt := t.TempDir()

	err := initNewProject(tgt, testFS)
	require.NoError(t, err)

	// Every entry in the source FS must exist in the target directory.
	err = fs.WalkDir(testFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "." {
			return err
		}
		tgtPath := filepath.Join(tgt, filepath.FromSlash(path))
		if d.IsDir() {
			assert.DirExists(t, tgtPath, "directory %q should be copied", path)
		} else {
			assert.FileExists(t, tgtPath, "file %q should be copied", path)
			want, _ := fs.ReadFile(testFS, path)
			got, _ := os.ReadFile(tgtPath)
			assert.Equal(t, want, got, "content of %q should match", path)
		}
		return nil
	})
	require.NoError(t, err)
}

func Test_initNewProject_ExistingDirIsOk(t *testing.T) {
	tgt := t.TempDir() // already exists

	err := initNewProject(tgt, testFS)
	require.NoError(t, err, "should succeed when target directory already exists")
}

func Test_initNewProject_FailsWhenTargetIsFile(t *testing.T) {
	// Create a regular file where the target dir should be.
	tmp := t.TempDir()
	tgt := filepath.Join(tmp, "notadir")
	require.NoError(t, os.WriteFile(tgt, []byte("data"), 0o644))

	err := initNewProject(tgt, testFS)
	require.Error(t, err, "should fail when target is a regular file")
}

// ─── runMCPNewProject ─────────────────────────────────────────────────────────

func Test_runMCPNewProject_UnknownLayout(t *testing.T) {
	err := runMCPNewProject(context.Background(), "nonexistent-layout", t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown project layout")
}

func Test_runMCPNewProject_KnownLayout(t *testing.T) {
	tgt := filepath.Join(t.TempDir(), "proj")
	// Use the real embedded assets for the known layout.
	err := runMCPNewProject(context.Background(), layoutOpencode, tgt)
	require.NoError(t, err)
	assert.DirExists(t, tgt)
}
