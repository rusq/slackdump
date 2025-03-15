package testutil

import (
	"io/fs"
	"testing"
)

type FileInfo struct {
	Name string
	Size int64
}

// CollectFiles returns a map of file paths to file info.
func CollectFiles(t *testing.T, fsys fs.FS) (ret map[string]FileInfo) {
	t.Helper()
	ret = make(map[string]FileInfo)
	if err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		ret[path] = FileInfo{Name: d.Name(), Size: fi.Size()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return
}
