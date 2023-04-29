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

// FSCloser is a FS that can be closed.
type FSCloser interface {
	FS
	io.Closer
}

// New returns appropriate filesystem based on the name of the location.
// Logic is simple:
//   - if location has a known extension, the appropriate adapter is returned.
//   - else: it's a directory.
//
// Currently supported extensions: ".zip" (case insensitive)
func New(location string) (FSCloser, error) {
	switch strings.ToUpper(filepath.Ext(location)) {
	case ".ZIP":
		return NewZipFile(location)
	default:
		return NewDirectory(location), nil
	}
}
