package chunk

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_hash(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"empty",
			args{""},
			"da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			"hello",
			args{"hello"},
			"aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hash(tt.args.s); got != tt.want {
				t.Errorf("hash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_hash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		hash("hello")
	}
}

func creategzfile(t *testing.T, dir string, name string, contents string) string {
	t.Helper()
	filename := filepath.Join(dir, name)
	f, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	if _, err := gz.Write([]byte(contents)); err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	return filename
}

func createfile(t *testing.T, dir string, name string, contents string) string {
	t.Helper()
	filename := filepath.Join(dir, name)
	f, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(contents); err != nil {
		t.Fatal(err)
	}
	return filename
}

func Test_filemgr_Open(t *testing.T) {
	// prepare some test files
	// workdir is the working directory.
	workdir := t.TempDir()

	// create compressed files
	creategzfile(t, workdir, "hello.gz", "hello")
	creategzfile(t, workdir, "world.gz", "world")

	type fields struct {
		// tmpdir  string // provided by t.TempDir()
		once    *sync.Once
		known   map[string]string
		handles map[string]io.Closer
	}
	type args struct {
		name string
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantContents string
		wantErr      bool
	}{
		{
			"hello",
			fields{
				once:    new(sync.Once),
				known:   make(map[string]string),
				handles: make(map[string]io.Closer),
			},
			args{filepath.Join(workdir, "hello.gz")},
			"hello",
			false,
		},
		{
			"world",
			fields{
				once:    new(sync.Once),
				known:   make(map[string]string),
				handles: make(map[string]io.Closer),
			},
			args{filepath.Join(workdir, "world.gz")},
			"world",
			false,
		},
		{
			"file not found",
			fields{
				once:    new(sync.Once),
				known:   make(map[string]string),
				handles: make(map[string]io.Closer),
			},
			args{filepath.Join(workdir, "notfound.gz")},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &filemgr{
				tmpdir:  t.TempDir(), // another temporary directory
				once:    tt.fields.once,
				known:   tt.fields.known,
				handles: tt.fields.handles,
			}
			got, err := dp.Open(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("filemgr.Open() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				defer got.Close()
				comparecontents(t, got, tt.wantContents)
			}
		})
	}
	t.Run("reopens the existing file", func(t *testing.T) {
		dp := &filemgr{
			tmpdir:  t.TempDir(), // another temporary directory
			once:    new(sync.Once),
			known:   make(map[string]string),
			handles: make(map[string]io.Closer),
		}
		f1, err := dp.Open(filepath.Join(workdir, "hello.gz"))
		if err != nil {
			t.Fatal(err)
		}
		comparecontents(t, f1, "hello")
		assert.Len(t, dp.handles, 1)
		f1.Close()
		assert.Len(t, dp.handles, 0)
		assert.Len(t, dp.known, 1)

		f2, err := dp.Open(filepath.Join(workdir, "hello.gz"))
		if err != nil {
			t.Fatal(err)
		}
		comparecontents(t, f2, "hello")
		assert.Len(t, dp.handles, 1)
		f2.Close()
		assert.Len(t, dp.handles, 0)
		assert.Len(t, dp.known, 1)

		f3, err := dp.Open(filepath.Join(workdir, "world.gz"))
		if err != nil {
			t.Fatal(err)
		}
		comparecontents(t, f3, "world")
		assert.Len(t, dp.handles, 1)
		f3.Close()
		assert.Len(t, dp.handles, 0)
		assert.Len(t, dp.known, 2)
	})
	t.Run("mkdirall fails", func(t *testing.T) {
		dp := &filemgr{
			tmpdir:  t.TempDir(), // another temporary directory
			once:    new(sync.Once),
			known:   make(map[string]string),
			handles: make(map[string]io.Closer),
		}
		os.Chmod(dp.tmpdir, 0o000)
		t.Cleanup(func() {
			os.Chmod(dp.tmpdir, 0o755)
		})
		dp.tmpdir = filepath.Join(dp.tmpdir, "non-existing")
		if _, err := dp.Open(filepath.Join(workdir, "hello.gz")); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("fails on non-compressed file", func(t *testing.T) {
		dp := &filemgr{
			tmpdir:  t.TempDir(), // another temporary directory
			once:    new(sync.Once),
			known:   make(map[string]string),
			handles: make(map[string]io.Closer),
		}
		filename := createfile(t, workdir, "hello.txt", "hello")
		if _, err := dp.Open(filename); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func comparecontents(t *testing.T, f *wrappedfile, want string) {
	t.Helper()
	buf, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, want, string(buf))
}

func Test_wrappedfile_Close(t *testing.T) {
	dir := t.TempDir()
	creategzfile(t, dir, "hello.gz", "hello")
	dp := &filemgr{
		tmpdir:  t.TempDir(), // another temporary directory
		once:    new(sync.Once),
		known:   make(map[string]string),
		handles: make(map[string]io.Closer),
	}
	f, err := dp.Open(filepath.Join(dir, "hello.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	assert.Len(t, dp.handles, 0)
	assert.Len(t, dp.known, 1)
	if assert.True(t, dp.mu.TryLock(), "mutex should be unlocked") {
		dp.mu.Unlock()
	}
}

func Test_filemgr_Destroy(t *testing.T) {
	t.Run("destroys the temporary directory", func(t *testing.T) {
		dp := &filemgr{
			tmpdir:  t.TempDir(),
			once:    new(sync.Once),
			known:   make(map[string]string),
			handles: make(map[string]io.Closer),
		}
		if err := dp.Destroy(); err != nil {
			t.Fatal(err)
		}
		_, err := os.Stat(dp.tmpdir)
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("closes all open file handles", func(t *testing.T) {
		dir := t.TempDir()
		creategzfile(t, dir, "hello.gz", "hello")
		creategzfile(t, dir, "world.gz", "world")
		dp := &filemgr{
			tmpdir:  t.TempDir(), // another temporary directory
			once:    new(sync.Once),
			known:   make(map[string]string),
			handles: make(map[string]io.Closer),
		}
		if _, err := dp.Open(filepath.Join(dir, "hello.gz")); err != nil {
			t.Fatal(err)
		}

		if _, err := dp.Open(filepath.Join(dir, "world.gz")); err != nil {
			t.Fatal(err)
		}
		assert.Len(t, dp.handles, 2)

		if err := dp.Destroy(); err != nil {
			t.Fatal(err)
		}

		assert.Len(t, dp.handles, 0)
		assert.Len(t, dp.known, 2)
		if assert.True(t, dp.mu.TryLock(), "mutex should be unlocked") {
			dp.mu.Unlock()
		}
	})
	t.Run("removeall called on non-existing directory", func(t *testing.T) {
		dp := &filemgr{
			tmpdir:  t.TempDir(),
			once:    new(sync.Once),
			known:   make(map[string]string),
			handles: make(map[string]io.Closer),
		}
		os.Chmod(dp.tmpdir, 0o000)
		t.Cleanup(func() {
			os.Chmod(dp.tmpdir, 0o755)
		})
		dp.tmpdir = filepath.Join(dp.tmpdir, "non-existing")
		if err := dp.Destroy(); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("errors closing file handles", func(t *testing.T) {
		dir := t.TempDir()
		creategzfile(t, dir, "hello.gz", "hello")
		dp := &filemgr{
			tmpdir:  t.TempDir(), // another temporary directory
			once:    new(sync.Once),
			known:   make(map[string]string),
			handles: make(map[string]io.Closer),
		}
		if _, err := dp.Open(filepath.Join(dir, "hello.gz")); err != nil {
			t.Fatal(err)
		}
		dp.handles["hello"] = &errcloser{err: assert.AnError}
		if err := dp.Destroy(); err == nil {
			t.Fatal("expected an error")
		}
	})
}

type errcloser struct {
	err error
}

func (ec *errcloser) Close() error {
	return ec.err
}
