package fsadapter

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"sync"
)

var _ FS = &ZIP{}

type ZIP struct {
	zw *zip.Writer
	mu sync.Mutex
	f  *os.File
}

func NewZIP(zw *zip.Writer) *ZIP {
	return &ZIP{zw: zw}
}

func NewZipFile(filename string) (*ZIP, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	zw := zip.NewWriter(f)
	return &ZIP{zw: zw, f: f}, nil
}

func (z *ZIP) Create(filepath string) (io.WriteCloser, error) {
	w, err := z.zw.Create(filepath)
	if err != nil {
		return nil, err
	}
	z.mu.Lock() // mutex will be unlocked, when the user calls Close.
	return &syncWriter{w: w, mu: &z.mu}, nil
}

func (z *ZIP) WriteFile(name string, data []byte, _ os.FileMode) error {
	z.mu.Lock()
	defer z.mu.Unlock()
	zf, err := z.zw.Create(name)
	if err != nil {
		return err
	}

	_, err = io.Copy(zf, bytes.NewReader(data))
	return err

}

// Close closes the underlying zip writer and the file handle.  It is only necessary if
// ZIP was initialised using NewZipFile
func (z *ZIP) Close() error {
	if !z.ours() {
		return nil
	}
	z.mu.Lock()
	if err := z.zw.Close(); err != nil {
		return err
	}
	if z.f == nil {
		return nil
	}
	return z.f.Close()
}

func (z *ZIP) ours() bool {
	return z.f != nil
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
