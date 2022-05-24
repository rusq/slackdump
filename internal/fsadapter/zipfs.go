package fsadapter

import (
	"archive/zip"
	"io"
	"sync"
)

var _ FileCreator = &ZIP{}

type ZIP struct {
	zw *zip.Writer
	mu sync.Mutex
}

func NewZIP(zw *zip.Writer) *ZIP {
	return &ZIP{zw: zw}
}

func (z *ZIP) Create(filepath string) (io.WriteCloser, error) {
	w, err := z.zw.Create(filepath)
	if err != nil {
		return nil, err
	}
	z.mu.Lock()
	return &syncWriter{w: w, mu: &z.mu}, nil
}

type syncWriter struct {
	w io.Writer // underlying writer

	// zip writer can only process one file at a time, so any process that wants
	// to Create the file will have to wait until Close is called:
	//
	// From zip.Create doc:  The file's contents must be written to the
	// io.Writer before the next call to Create, CreateHeader, or Close.
	mu *sync.Mutex
}

func (sw *syncWriter) Write(p []byte) (int, error) {
	return sw.w.Write(p)
}

func (sw *syncWriter) Close() error {
	sw.mu.Unlock()
	return nil
}
