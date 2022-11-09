package fsadapter

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FS is interface for operating on the files of the underlying filesystem.
type FS interface {
	Create(string) (io.WriteCloser, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// New returns appropriate filesystem based on the name of the location.
// Logic is simple:
//   - if location has a known extension, the appropriate adapter is returned.
//   - else: it's a directory.
//
// Currently supported extensions: ".zip" (case insensitive)
func New(location string) (FS, error) {
	switch strings.ToUpper(filepath.Ext(location)) {
	case ".ZIP":
		return NewZipFile(location)
	default:
		return NewDirectory(location), nil
	}
}

// Close closes the filesystem, if it implements the io.Closer interface.
func Close(fs FS) error {
	closer, ok := fs.(io.Closer)
	if !ok {
		return nil
	}
	return closer.Close()
}
