package fsadapter

import (
	"io"
	"os"
)

// FS is interface for operating on the files of the underlying filesystem.
type FS interface {
	Create(string) (io.WriteCloser, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// Close closes the filesystem, if it implements the io.Closer interface.
func Close(fs FS) error {
	closer, ok := fs.(io.Closer)
	if !ok {
		return nil
	}
	return closer.Close()
}
