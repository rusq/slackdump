package osext

import (
	"compress/gzip"
	"io"
	"os"
)

const tempMask = "osext-*"

// UnGZIP decompresses a gzip file and returns a temporary file handler.
// it must be removed after use.  It expects r to contain a gzip file data.
func UnGZIP(r io.Reader) (*os.File, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	f, err := os.CreateTemp("", tempMask)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(f, gr)
	if err != nil {
		return nil, err
	}
	if err := f.Sync(); err != nil {
		return nil, err
	}
	// reset temporary file position to prepare it for reading.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return f, nil
}
