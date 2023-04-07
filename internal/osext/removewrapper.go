package osext

import (
	"io"
	"os"
)

// RemoveOnClose wraps an io.ReadSeekCloser and removes the file when it is
// closed.  The filename must be given.
func RemoveOnClose(r io.ReadSeekCloser, filename string) io.ReadSeekCloser {
	return RemoveWrapper{filename: filename, ReadSeekCloser: r}
}

type RemoveWrapper struct {
	io.ReadSeekCloser

	filename string
}

func (r RemoveWrapper) Close() error {
	err := r.ReadSeekCloser.Close()
	if err != nil {
		return err
	}
	return os.Remove(r.filename)
}

func (r RemoveWrapper) Filename() string {
	return r.filename
}
