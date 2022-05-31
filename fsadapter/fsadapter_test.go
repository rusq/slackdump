package fsadapter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeCloser struct {
	fakeFS
}

type fakeFS struct {
	isClosed bool
}

func (c *fakeCloser) Close() error {
	c.fakeFS.isClosed = true
	return nil
}

func (c *fakeFS) Create(_ string) (io.WriteCloser, error) {
	return nil, nil
}

func (c *fakeFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return nil
}

func TestClose(t *testing.T) {
	clfs := new(fakeCloser)
	if err := Close(clfs); err != nil {
		t.Error("that is completely unexpected")
	}
	assert.True(t, clfs.isClosed)

	ffs := new(fakeFS)
	if err := Close(ffs); err != nil {
		t.Errorf("that is even more unexpected")
	}
	assert.False(t, ffs.isClosed)
}

func TestForFilename(t *testing.T) {
	tmp := t.TempDir()
	type args struct {
		name string
	}
	tests := []struct {
		name       string
		args       args
		wantString string
		wantErr    bool
	}{
		{
			"directory",
			args{filepath.Join(tmp, "blah")},
			"<directory: " + filepath.Join(tmp, "blah") + ">",
			false,
		},
		{
			"zip file",
			args{filepath.Join(tmp, "bloop.zip")},
			"<zip archive: " + filepath.Join(tmp, "bloop.zip") + ">",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ForFilename(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ForFilename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer Close(got)

			assert.Equal(t, tt.wantString, fmt.Sprint(got))
		})
	}
}
