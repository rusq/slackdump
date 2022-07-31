// Package encio provides encrypted input/output functions
package encio

import (
	"bytes"
	"crypto/aes"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

const plaintext = "1234567890123456"

func TestNewReadWriter(t *testing.T) {
	var buf bytes.Buffer

	w, err := NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}

	n, err := w.Write([]byte(plaintext))
	if err != nil {
		t.Fatalf("error encrypting text: %s", err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("error closing writer: %s", err)
	}
	if n != len(plaintext) {
		t.Errorf("incosistent write byte count: want=%d, got=%d", len(plaintext), n)
	}
	if sz := len(plaintext) + aes.BlockSize; sz != buf.Len() {
		t.Errorf("invalid encrypted message size: want=%d, got=%d", sz, buf.Len())
	}

	r, err := NewReader(&buf)
	if err != nil {
		t.Errorf("error creating reader: %s", err)
	}
	var result strings.Builder
	if _, err := io.Copy(&result, r); err != nil {
		t.Errorf("error reading encrypted data: %s", err)
	}
	if !strings.EqualFold(result.String(), plaintext) {
		t.Errorf("invalid decrypted text: want=%q, got=%q", plaintext, result.String())
	}
}

func TestCreateOpen(t *testing.T) {
	const testfile = "testfile"
	td := t.TempDir()
	f, err := Create(filepath.Join(td, testfile))
	if err != nil {
		t.Fatalf("error creating a test file: %s", err)
	}
	if n, err := f.Write([]byte(plaintext)); err != nil {
		t.Errorf("error writing test data: %s", err)
	} else if n != len(plaintext) {
		t.Errorf("unexpected number of bytes written: want=%d, got=%d", len(plaintext), n)
	}
	if err := f.Close(); err != nil {
		t.Errorf("error while closing R/W file: %s", err)
	}

	g, err := Open(filepath.Join(td, testfile))
	if err != nil {
		t.Errorf("error opening a test file: %s", err)
	}
	defer func() {
		if err := g.Close(); err != nil {
			t.Errorf("error while closing R/O file: %s", err)
		}
	}()

	var result strings.Builder
	if _, err := io.Copy(&result, g); err != nil {
		t.Errorf("error reading encrypted data: %s", err)
	}
	if !strings.EqualFold(result.String(), plaintext) {
		t.Errorf("invalid decrypted text: want=%q, got=%q", plaintext, result.String())
	}
}

func TestSetAppID(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"set app ID",
			args{"test"},
			false,
		},
		{
			"empty app ID",
			args{""},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldAppID := appID
			defer func() {
				appID = oldAppID
			}()
			err := SetAppID(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetAppID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if appID != tt.args.s {
				t.Errorf("SetAppID failed to set the appID. want=%q, got=%q", tt.args.s, appID)
			}
		})
	}
}

func Test_generateIV(t *testing.T) {
	iv1, err := generateIV()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	iv2, err2 := generateIV()
	if err2 != nil {
		t.Errorf("unexpected error 2: %s", err2)
	}

	if bytes.EqualFold(iv1[:], iv2[:]) {
		t.Errorf("same IV was generated, while it must be random: iv1=%#v, iv2=%#v", iv1, iv2)
	}
}

type fakeCloser struct {
	closeCalledTimes int
}

func (c *fakeCloser) Close() error {
	c.closeCalledTimes++
	return nil
}

type fakeReadWriter struct {
	fakeCloser

	readCalledTimes  int
	writeCalledTimes int
}

func (frw *fakeReadWriter) Read(b []byte) (int64, error) {
	frw.readCalledTimes++
	return int64(len(b)), nil
}

func (frw *fakeReadWriter) Write(b []byte) (int, error) {
	frw.writeCalledTimes++
	return len(b), nil
}

func Test_writeCloser_Close(t *testing.T) {
	var (
		frw = new(fakeReadWriter)
		fc  = new(fakeCloser)
	)

	fw := writeCloser{
		f:           fc,
		WriteCloser: frw,
	}

	fw.Close()

	if fc.closeCalledTimes != 1 {
		t.Errorf("f.Close called unexpected number of times: %d", 1)
	}
	if frw.closeCalledTimes != 1 {
		t.Errorf("wc.Close called unexpected number of times: %d", 1)
	}
}
