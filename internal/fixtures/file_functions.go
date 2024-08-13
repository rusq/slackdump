package fixtures

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MkTestFile creates a test file in the directory dir, and copies the content
// into it.
func MkTestFile(t *testing.T, dir string, content string) string {
	t.Helper()

	f, err := os.CreateTemp(dir, "")
	if err != nil {
		t.Fatal("create temp:", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, strings.NewReader(content)); err != nil {
		t.Fatal("copy:", err)
	}
	return f.Name()
}

// MkTestFileName creates a test file at the path, and copies the content into
// it.
func MkTestFileName(t *testing.T, path, content string) string {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal("write file:", err)
	}

	return path
}

// PrepareDir creates a directory with test files.
func PrepareDir(t *testing.T, dir string, content string, files ...string) {
	t.Helper()
	for _, filename := range JoinPath(dir, files...) {
		if err := os.WriteFile(filename, []byte("dummy"), 0600); err != nil {
			t.Fatalf("error writing %q: %s", filename, err)
		}
	}
}

var WorkspaceFiles = []string{"ora600.bin", "sdump.bin", "foo.bin", "bar.bin", "provider.bin", "default.bin"}

// JoinPath joins the dir to each file in files using filepath.Join.
func JoinPath(dir string, files ...string) []string {
	ff := make([]string, 0, len(files))
	for _, filename := range files {
		ff = append(ff, filepath.Join(dir, filename))
	}
	return ff
}

// StripExt strips file extension.
func StripExt(filename string) string {
	if len(filename) == 0 {
		return filename
	}
	if len(filepath.Ext(filename)) == 0 {
		return filename
	}
	return filename[0 : len(filename)-len(filepath.Ext(filename))]
}
