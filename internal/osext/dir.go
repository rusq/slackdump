package osext

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNotADir is returned when the path is not a directory.
var ErrNotADir = errors.New("not a directory")

// DirExists checks if the directory exists and is a directory.  It will return
// an error if the path does not exist, and if the path is not a directory,
// ErrNotADir will be returned.
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

// IsSame returns true if path1 and path2 both pointing to the same object.
func IsSame(path1, path2 string) (bool, error) {
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
