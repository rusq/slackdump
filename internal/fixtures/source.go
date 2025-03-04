package fixtures

import (
	"archive/zip"
	"bytes"
	"embed"
	"io/fs"
	"path/filepath"
	"testing"
)

//go:embed assets/source_dump_dir
var fsTestDumpDir embed.FS

//go:embed assets/source_dump.zip
var fsTestDumpZIP []byte

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

var testDumpDirPath = filepath.Join("assets", "source_dump_dir")

func init() {
	FSTestDumpDir = must(fs.Sub(fsTestDumpDir, testDumpDirPath))
}

// FSTempDumpDir is a filesystem with a dump directory.
var FSTestDumpDir fs.FS

// FSTestDumpZIP returns a filesystem of Dump ZIP archive.
func FSTestDumpZIP(t *testing.T) fs.FS {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(fsTestDumpZIP), int64(len(fsTestDumpZIP)))
	if err != nil {
		t.Fatal(err)
	}
	return zr
}
