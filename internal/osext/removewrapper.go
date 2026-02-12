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
	"io"
	"os"
)

// ReadSeekCloseNamer is an io.ReadSeekCloser that also has a Name method.
type ReadSeekCloseNamer interface {
	io.ReadSeekCloser
	Name() string
}

// RemoveOnClose wraps an *os.File and removes it when it is closed.  The
// filename must be given.
func RemoveOnClose(r *os.File) ReadSeekCloseNamer {
	return RemoveWrapper{filename: r.Name(), ReadSeekCloser: r}
}

// RemoveWrapper wraps an io.ReadSeekCloser and removes the file when it is
// closed.
type RemoveWrapper struct {
	io.ReadSeekCloser

	filename string
}

func (r RemoveWrapper) Close() error {
	err := r.ReadSeekCloser.Close()
	if err != nil {
		return err
	}
	return os.Remove(r.filename)
}

func (r RemoveWrapper) Name() string {
	return r.filename
}
