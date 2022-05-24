package fsadapter

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

var _ FileCreator = Filesystem{}

type Filesystem struct {
	dir string
}

func NewFilesystem(dir string) Filesystem {
	return Filesystem{dir: dir}
}

func (fs Filesystem) Create(fpath string) (io.WriteCloser, error) {
	node := filepath.Join(fs.dir, fpath)
	nodeDir := filepath.Dir(node)
	if err := mkdir(nodeDir); err != nil {
		return nil, err
	}
	return os.Create(node)
}

// mkdir creates a directory "name", if the directory exists, it does nothing.
func mkdir(name string) error {
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
