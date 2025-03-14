package convert

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rusq/fsadapter"
)

func Test_copy2trg(t *testing.T) {
	t.Run("copy ok", func(t *testing.T) {
		srcdir := t.TempDir()
		trgdir := t.TempDir()

		if err := os.WriteFile(filepath.Join(srcdir, "test.txt"), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
		trgfs := fsadapter.NewDirectory(trgdir)
		srcfs := os.DirFS(srcdir)

		if err := copy2trg(trgfs, "test-copy.txt", srcfs, "test.txt"); err != nil {
			t.Fatal(err)
		}
		// validate
		data, err := os.ReadFile(filepath.Join(trgdir, "test-copy.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "test" {
			t.Fatal("unexpected data")
		}
	})
	t.Run("copy fails", func(t *testing.T) {
		srcdir := t.TempDir()
		trgdir := t.TempDir()

		srcfs := os.DirFS(srcdir)
		trgfs := fsadapter.NewDirectory(trgdir)
		// source file does not exist.
		if err := copy2trg(trgfs, "test-copy.txt", srcfs, "test.txt"); err == nil {
			t.Fatal("expected error, but got nil")
		}
	})
}
