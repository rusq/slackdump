// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
