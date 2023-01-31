package fsadapter

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FS is interface for operating on the files of the underlying filesystem.
//
//go:generate mockgen -destination ../internal/mocks/mock_fsadapter/mock_fs.go github.com/rusq/slackdump/v2/fsadapter FS
type FS interface {
	Create(string) (io.WriteCloser, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
}

// ForFilename returns appropriate filesystem based on the name of the file or
// directory given.
// Logic is simple:
//   - if file has a known extension, the appropriate adapter will be returned.
//   - else: it's a directory.
func ForFilename(name string) (FS, error) {
	switch strings.ToUpper(filepath.Ext(name)) {
	case ".ZIP":
		return NewZipFile(name)
	default:
		return NewDirectory(name), nil
	}
}

// Close closes the filesystem, if it implements the io.Closer interface.
func Close(fs FS) error {
	closer, ok := fs.(io.Closer)
	if !ok {
		return nil
	}
	return closer.Close()
}
