package fsadapter

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
)

var _ FS = &ZIP{}

type ZIP struct {
	zw *zip.Writer
	mu sync.Mutex
	f  *os.File
}

func (z *ZIP) String() string {
	return fmt.Sprintf("<zip archive: %s>", z.f.Name())
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

func (*ZIP) normalizePath(p string) string {
	return path.Join(filepath.SplitList(filepath.Clean(p))...)
}

func (z *ZIP) Create(filename string) (io.WriteCloser, error) {
	// reassemble path in correct format for ZIP file
	// in case it uses OS specific path.
	filename = z.normalizePath(filename)

	z.mu.Lock() // mutex will be unlocked, when the user calls Close.
	w, err := z.zw.Create(filename)
	if err != nil {
		return nil, err
	}
	return &syncWriter{w: w, mu: &z.mu}, nil
}

func (z *ZIP) WriteFile(filename string, data []byte, _ os.FileMode) error {
	z.mu.Lock()
	defer z.mu.Unlock()
	zf, err := z.zw.Create(filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(zf, bytes.NewReader(data))
	return err

}

// Close closes the underlying zip writer and the file handle.  It is only necessary if
// ZIP was initialised using NewZipFile
func (z *ZIP) Close() error {
	if !z.ourHandles() {
		// we don't own the handles, so just bail out.
		return nil
	}
	z.mu.Lock()
	defer z.mu.Unlock()

	return z.closeHandles()
}

func (z *ZIP) closeHandles() error {
	if err := z.zw.Close(); err != nil {
		return err
	}
	if z.f == nil {
		return nil
	}
	return z.f.Close()
}

func (z *ZIP) ourHandles() bool {
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
