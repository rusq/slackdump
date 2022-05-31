package fsadapter

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewZipFile(t *testing.T) {
	tmp := t.TempDir()
	t.Run("zip file is created", func(t *testing.T) {
		zipPath := filepath.Join(tmp, "test.zip")
		zf, err := NewZipFile(zipPath)
		require.NoError(t, err)

		assert.NotNil(t, zf.zw)
		assert.NotNil(t, zf.f)

		require.NoError(t, zf.Close())
		require.FileExists(t, zipPath)
	})
	t.Run("error creating a file", func(t *testing.T) {
		zipPath := tmp // should fail on directory
		_, err := NewZipFile(zipPath)
		require.Error(t, err)
	})
}

func TestZIP_Create(t *testing.T) {
	tmp := t.TempDir()
	t.Run("file is created", func(t *testing.T) {
		testsuiteZipFile(t, filepath.Join(tmp, "test.zip"), "abc/def.txt", "abcdef")
	})
	t.Run("backslashes", func(t *testing.T) {
		testsuiteZipFile(t, filepath.Join(tmp, "test.zip"), "abc\\backslash.txt", "abcdef")
	})
}

func testsuiteZipFile(t *testing.T, zipFile, filename, content string) {
	hArc, err := os.Create(zipFile)
	require.NoError(t, err)
	defer hArc.Close()
	zw := zip.NewWriter(hArc)
	z := ZIP{zw: zw}

	// create test file
	hTest, err := z.Create(filename)
	require.NoError(t, err)
	_, err = io.Copy(hTest, strings.NewReader(content))
	assert.NoError(t, err)
	assert.NoError(t, z.Close())
	// test file closed

	assert.NoError(t, zw.Close())
	assert.NoError(t, hArc.Close())

	assertZippedFileSize(t, zipFile, filename, uint64(len(content)))
}

func assertZippedFileSize(t *testing.T, zipfile string, fullpath string, size uint64) {
	zr, err := zip.OpenReader(zipfile)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	zzz := ZIP{}

	found := false
	for _, f := range zr.File {
		t.Log(f.Name)
		if f.Name == zzz.normalizePath(fullpath) {
			if f.UncompressedSize64 != size {
				t.Errorf("file: %s, size mismatch: want: %d, got %d", fullpath, size, f.UncompressedSize64)
				return
			}
			t.Log("size OK")
			return
		}
	}
	if !found {
		t.Errorf("file not found: %s", fullpath)
	}
}
