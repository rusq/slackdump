package fsadapter

import (
	"io"
	"os"
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
