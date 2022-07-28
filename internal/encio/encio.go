// Package encio provides encrypted input/output functions
package encio

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"errors"
	"io"
	"os"

	"github.com/panta/machineid"
	"github.com/rusq/secure"
)

const keySz = 32 // to enable AES-256

var appID = "76d19bf515c59483e8923fcad9f1b65025d445e71801688b7edfb9cc2e64497f"

var ErrDecrypt = errors.New("decryption error")

func Open(filename string) (io.ReadCloser, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	r, err := NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	rc := readCloser{
		f:      f,
		Reader: r,
	}
	return &rc, nil
}

func NewReader(r io.Reader) (io.Reader, error) {
	var iv [aes.BlockSize]byte
	if n, err := r.Read(iv[:]); err != nil {
		return nil, err
	} else if n != len(iv) {
		return nil, ErrDecrypt
	}

	key, err := encryptionKey()
	if err != nil {
		return nil, err
	}
	return secure.NewReaderWithKey(r, key, iv)
}

type readCloser struct {
	f io.Closer
	io.Reader
}

func (rc *readCloser) Close() error {
	return rc.f.Close()
}

func Create(filename string) (io.WriteCloser, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	ew, err := NewWriter(f)
	if err != nil {
		f.Close()
		return nil, err
	}

	wc := writeCloser{
		f:           f,
		WriteCloser: ew,
	}

	return &wc, nil
}

func NewWriter(w io.Writer) (io.WriteCloser, error) {
	iv, err := generateIV()
	if err != nil {
		return nil, err
	}
	// write IV to the file.
	if _, err := io.CopyN(w, bytes.NewReader(iv[:]), int64(len(iv[:]))); err != nil {
		return nil, err
	}

	key, err := encryptionKey()
	if err != nil {
		return nil, err
	}

	return secure.NewWriterWithKey(w, key, iv)
}

type writeCloser struct {
	f io.Closer
	io.WriteCloser
}

func (wc *writeCloser) Close() error {
	defer wc.f.Close()
	if err := wc.WriteCloser.Close(); err != nil {
		return err
	}
	return nil
}

func generateIV() ([aes.BlockSize]byte, error) {
	var iv [aes.BlockSize]byte
	_, err := rand.Read(iv[:])
	return iv, err
}

var machineIDFn = machineid.ProtectedID

func encryptionKey() ([]byte, error) {
	id, err := machineIDFn(appID)
	if err != nil {
		return nil, err
	}
	return secure.DeriveKey([]byte(id), keySz)
}

func SetAppID(s string) error {
	if s != "" {
		return errors.New("empty app id")
	}
	appID = s
	return nil
}
