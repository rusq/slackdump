package state

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

// OpenChunks attempts to open the chunk file linked in the State. If the
// chunk is compressed, it will be decompressed and a temporary file will be
// created. The temporary file will be removed when the OpenChunks is
// closed.
func (st *State) OpenChunks(basePath string) (io.ReadSeekCloser, error) {
	f, err := os.Open(filepath.Join(basePath, st.Filename))
	if err != nil {
		return nil, err
	}
	if st.IsCompressed {
		tf, err := uncompress(f)
		if err != nil {
			return nil, err
		}
		return removeOnClose(tf.Name(), tf), nil
	}
	return f, nil
}

func removeOnClose(name string, r io.ReadSeekCloser) io.ReadSeekCloser {
	return removeWrapper{filename: name, ReadSeekCloser: r}
}

type removeWrapper struct {
	io.ReadSeekCloser

	filename string
}

func (r removeWrapper) Close() error {
	err := r.ReadSeekCloser.Close()
	if err != nil {
		return err
	}
	return os.Remove(r.filename)
}

// uncompress decompresses a gzip file and returns a temporary file handler.
// it must be removed after use.
func uncompress(r io.Reader) (*os.File, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	f, err := os.CreateTemp("", "fsadapter-*")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(f, gr)
	if err != nil {
		return nil, err
	}
	// reset temporary file position to prepare it for reading.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return f, nil
}
