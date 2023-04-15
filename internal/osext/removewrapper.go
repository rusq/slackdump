package osext

import (
	"io"
	"os"
)

// RemoveOnClose wraps an *os.File and removes it when it is closed.  The
// filename must be given.
func RemoveOnClose(r *os.File) io.ReadSeekCloser {
	return RemoveWrapper{filename: r.Name(), ReadSeekCloser: r}
}

// RemoveWrapper wraps an io.ReadSeekCloser and removes the file when it is
// closed.
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
