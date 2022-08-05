// Package encio provides encrypted using AES-256-CFB input/output functions.
//
// Encrypted container structure is the following:
//
//   |__...__|____________...
//    0  ^   16   ^
//       |        +-- encrypted data
//       +----------- 16 bytes IV
//
package encio

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/rusq/secure"
)

const keySz = 32 // 32 bytes key size enables the AES-256

var appID = "76d19bf515c59483e8923fcad9f1b65025d445e71801688b7edfb9cc2e64497f"

var ErrDecrypt = errors.New("decryption error")

// Open opens an encrypted file container.
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

// NewReader wraps the ciphertext reader, and returns the reader that a
// plaintext can be read from.
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

// readCloser wraps around the file closer and the reader.
type readCloser struct {
	f io.Closer
	io.Reader
}

// Close closes the underlying file.
func (rc *readCloser) Close() error {
	return rc.f.Close()
}

// Create creates an encrypted file container.
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

// NewWriter wraps the writer and returns the WriteCloser.  Any information
// written to the writer is encrypted with the hashed machineID.  WriteCloser
// must be closed to flush any buffered data.
func NewWriter(w io.Writer) (io.WriteCloser, error) {
	iv, err := generateIV()
	if err != nil {
		return nil, err
	}
	// write IV to the file.
	if _, err := io.CopyN(w, bytes.NewReader(iv[:]), int64(len(iv[:]))); err != nil {
		return nil, fmt.Errorf("failed to write the initialisation vector: %w", err)
	}

	key, err := encryptionKey()
	if err != nil {
		return nil, err
	}

	return secure.NewWriterWithKey(w, key, iv)
}

// writeCloser is a wrapper around file closer and the cipher WriteCloser.
type writeCloser struct {
	f io.Closer
	io.WriteCloser
}

// Close closes the encrypted Writer and the underlying file.
func (wc *writeCloser) Close() error {
	defer wc.f.Close()
	if err := wc.WriteCloser.Close(); err != nil {
		return err
	}
	return nil
}

// generateIV generates the random initialisation vector.
func generateIV() ([aes.BlockSize]byte, error) {
	var iv [aes.BlockSize]byte
	_, err := io.ReadFull(rand.Reader, iv[:])
	return iv, err
}

// encryptionKey returns an encryption key from the passphrase that is
// generated from a hashed by appID machineID.
func encryptionKey() ([]byte, error) {
	id, err := machineIDFn(appID)
	if err != nil {
		return nil, err
	}
	return secure.DeriveKey([]byte(id), keySz)
}

// SetAppID allows to set the appID, that is used to hash the value of
// machineID.
func SetAppID(s string) error {
	if s == "" {
		return errors.New("empty app id")
	}
	appID = s
	return nil
}
