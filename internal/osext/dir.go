package osext

import (
	"errors"
	"os"
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
