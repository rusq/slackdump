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
	"errors"
	"os"
	"path/filepath"
)

// ErrNotADir is returned when the path is not a directory.
var ErrNotADir = errors.New("not a directory")

// DirExists checks if the directory exists and is a directory.  It will return
// an error if the path does not exist, and if the path is not a directory,
// ErrNotADir will be returned.
func DirExists(dir string) error {
	fi, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return ErrNotADir
	}
	return nil
}

// IsSame returns true if path1 and path2 both pointing to the same object.
func IsSame(path1, path2 string) (bool, error) {
	ap1, err := filepath.Abs(path1)
	if err != nil {
		return false, err
	}
	ap2, err := filepath.Abs(path2)
	if err != nil {
		return false, err
	}
	return ap1 == ap2, nil
}
