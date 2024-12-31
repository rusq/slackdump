package osext

import (
	"fmt"
	"io"
	"os"

	"github.com/rusq/fsadapter"
)

// MoveFile moves a file from src to dst.  If dst already exists, it will be
// overwritten.
//
// Adopted solution from https://stackoverflow.com/questions/50740902/move-a-file-to-a-different-drive-with-go
func MoveFile(src string, fs fsadapter.FS, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("unable to open source file: %s", err)
	}

	out, err := fs.Create(dst)
	if err != nil {
		in.Close()
		return fmt.Errorf("unable to open destination file: %s", err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	in.Close()
	if err != nil {
		return fmt.Errorf("error writing output: %s", err)
	}

	// sync is not supported by fsadapter.
	// if err := out.Sync(); err != nil {
	// 	return fmt.Errorf("sync: %s", err)
	// }

	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("stat: %s", err)
	} else {
		// Chmod not yet supported.
		// if err := fs.Chmod(dst, si.Mode()); err != nil {
		// 	return fmt.Errorf("chmod: %s", err)
		// }
		_ = err // ignore SA9003 in golang-ci-lint
	}

	if err := os.Remove(src); err != nil {
		return fmt.Errorf("failed removing source: %s", err)
	}
	return nil
}
