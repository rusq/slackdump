package fsadapter

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

var _ FS = Directory{}

type Directory struct {
	dir string
}

func NewDirectory(dir string) Directory {
	return Directory{dir: dir}
}

func (d Directory) String() string {
	return "<directory: " + d.dir + ">"
}

func (fs Directory) Create(fpath string) (io.WriteCloser, error) {
	node := filepath.Join(fs.dir, fpath)
	nodeDir := filepath.Dir(node)
	if err := mkdirAll(nodeDir); err != nil {
		return nil, err
	}
	return os.Create(node)
}

// mkdirAll creates a directory "name", if the directory exists, it does nothing.
func mkdirAll(name string) error {
	if name == "" {
		return errors.New("empty directory")
	}

	fi, err := os.Stat(name)
	if err == nil && fi.IsDir() {
		// exists and is a directory
		return nil
	}

	if err := os.MkdirAll(name, 0755); err != nil {
		return err
	}
	return nil
}

func (fs Directory) WriteFile(name string, data []byte, perm os.FileMode) error {
	fullpath := filepath.Join(fs.dir, name)
	if err := mkdirAll(filepath.Dir(fullpath)); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(fs.dir, name), data, perm)
}
