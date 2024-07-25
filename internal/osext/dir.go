package osext

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrNotADir = errors.New("not a directory")

func DirExists(dir string) error {
	fi, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return ErrNotADir
	}
	return nil
}

func Same(path1, path2 string) (bool, error) {
	ap1, err := filepath.Abs(path1)
	if err != nil {
		return false, err
	}
	ap2, err := filepath.Abs(path2)
	if err != nil {
		return false, err
	}
	return ap1 == ap2, nil
}
