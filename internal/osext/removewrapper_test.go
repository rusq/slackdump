package osext

import (
	"os"
	"testing"
)

func TestRemoveOnClose(t *testing.T) {
	d := t.TempDir()
	t.Run("removes the file on close", func(t *testing.T) {
		f, err := os.CreateTemp(d, "test")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		r := RemoveOnClose(f)
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(f.Name()); !os.IsNotExist(err) {
			t.Errorf("file %s still exists", f.Name())
		}
	})
}

func TestRemoveWrapper_Name(t *testing.T) {
	d := t.TempDir()
	t.Run("returns the filename", func(t *testing.T) {
		f, err := os.CreateTemp(d, "test")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		r := RemoveOnClose(f)
		if r.Name() != f.Name() {
			t.Errorf("Name() = %s, want %s", r.Name(), f.Name())
		}
	})
}
