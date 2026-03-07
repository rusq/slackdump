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
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLayoutFS is a minimal layout FS: one config file and a layout.json
// manifest that references two skills.
var testLayoutFS = fstest.MapFS{
	"layout.json": {Data: []byte(`{
		"files":  [{"src":"config.json","dst":"config.json"}],
		"skills": [
			{"skill":"skill-a","dst":"dest/skill-a.md"},
			{"skill":"skill-b","dst":"dest/skill-b.md"}
		]
	}`)},
	"config.json": {Data: []byte(`{"mcp":{}}`)},
}

// testSkillsFS is a minimal shared-skills FS.
var testSkillsFS = fstest.MapFS{
	"skill-a/SKILL.md": {Data: []byte("# skill-a\n")},
	"skill-b/SKILL.md": {Data: []byte("# skill-b\n")},
}

// ─── readManifest ─────────────────────────────────────────────────────────────

func Test_readManifest_ParsesOk(t *testing.T) {
	m, err := readManifest(testLayoutFS)
	require.NoError(t, err)

	require.Len(t, m.Files, 1)
	assert.Equal(t, "config.json", m.Files[0].Src)
	assert.Equal(t, "config.json", m.Files[0].Dst)

	require.Len(t, m.Skills, 2)
	assert.Equal(t, "skill-a", m.Skills[0].Skill)
	assert.Equal(t, "dest/skill-a.md", m.Skills[0].Dst)
}

func Test_readManifest_MissingFile(t *testing.T) {
	_, err := readManifest(fstest.MapFS{})
	require.Error(t, err)
}

// ─── initNewProject ───────────────────────────────────────────────────────────

func Test_initNewProject_CreatesTargetDir(t *testing.T) {
	tgt := filepath.Join(t.TempDir(), "new-project")
	require.NoDirExists(t, tgt)

	err := initNewProject(tgt, testLayoutFS, testSkillsFS)
	require.NoError(t, err)

	assert.DirExists(t, tgt)
}

func Test_initNewProject_CopiesLayoutFiles(t *testing.T) {
	tgt := t.TempDir()

	err := initNewProject(tgt, testLayoutFS, testSkillsFS)
	require.NoError(t, err)

	// layout-specific file
	assertFileContent(t, filepath.Join(tgt, "config.json"), `{"mcp":{}}`)
}

func Test_initNewProject_CopiesSkills(t *testing.T) {
	tgt := t.TempDir()

	err := initNewProject(tgt, testLayoutFS, testSkillsFS)
	require.NoError(t, err)

	assertFileContent(t, filepath.Join(tgt, "dest", "skill-a.md"), "# skill-a\n")
	assertFileContent(t, filepath.Join(tgt, "dest", "skill-b.md"), "# skill-b\n")
}

func Test_initNewProject_ExistingDirIsOk(t *testing.T) {
	tgt := t.TempDir()

	err := initNewProject(tgt, testLayoutFS, testSkillsFS)
	require.NoError(t, err, "should succeed when target directory already exists")
}

func Test_initNewProject_FailsWhenTargetIsFile(t *testing.T) {
	tmp := t.TempDir()
	tgt := filepath.Join(tmp, "notadir")
	require.NoError(t, os.WriteFile(tgt, []byte("data"), 0o644))

	err := initNewProject(tgt, testLayoutFS, testSkillsFS)
	require.Error(t, err, "should fail when target is a regular file")
}

func Test_initNewProject_MissingSkill(t *testing.T) {
	layoutFS := fstest.MapFS{
		"layout.json": {Data: []byte(`{"skills":[{"skill":"missing","dst":"out.md"}]}`)},
	}
	err := initNewProject(t.TempDir(), layoutFS, fstest.MapFS{})
	require.Error(t, err)
}

// ─── runMCPNewProject ─────────────────────────────────────────────────────────

func Test_runMCPNewProject_UnknownLayout(t *testing.T) {
	err := runMCPNewProject(context.Background(), "nonexistent-layout", t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown project layout")
}

func Test_runMCPNewProject_Opencode(t *testing.T) {
	tgt := filepath.Join(t.TempDir(), "proj")
	err := runMCPNewProject(context.Background(), layoutOpencode, tgt)
	require.NoError(t, err)
	assert.DirExists(t, tgt)
	assert.FileExists(t, filepath.Join(tgt, "opencode.jsonc"))
	assert.FileExists(t, filepath.Join(tgt, ".opencode", "skills", "slackdump", "SKILL.md"))
}

func Test_runMCPNewProject_ClaudeCode(t *testing.T) {
	tgt := filepath.Join(t.TempDir(), "proj")
	err := runMCPNewProject(context.Background(), layoutClaudeCode, tgt)
	require.NoError(t, err)
	assert.DirExists(t, tgt)
	assert.FileExists(t, filepath.Join(tgt, ".mcp.json"))
	assert.FileExists(t, filepath.Join(tgt, "CLAUDE.md"))
}

func Test_runMCPNewProject_Copilot(t *testing.T) {
	tgt := filepath.Join(t.TempDir(), "proj")
	err := runMCPNewProject(context.Background(), layoutCopilot, tgt)
	require.NoError(t, err)
	assert.DirExists(t, tgt)
	assert.FileExists(t, filepath.Join(tgt, ".vscode", "mcp.json"))
	assert.FileExists(t, filepath.Join(tgt, ".github", "copilot-instructions.md"))
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	require.FileExists(t, path)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
}
