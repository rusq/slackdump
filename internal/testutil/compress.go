package testutil

import (
	"bytes"
	"compress/gzip"
	"testing"
)

// GZCompress compresses data using gzip and returns the compressed data.
func GZCompress(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	defer gz.Close()
	if _, err := gz.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
