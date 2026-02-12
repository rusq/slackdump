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

// Package osext provides some helpful os functions.
package osext

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Error struct {
	File string
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Err, e.File)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Namer is an interface that allows us to get the name of the file.
type Namer interface {
	// Name should return the name of the file.  *os.File implements this
	// interface.
	Name() string
}

func Caller(steps int) string {
	name := "?"
	if pc, _, _, ok := runtime.Caller(steps + 1); ok {
		name = filepath.Base(runtime.FuncForPC(pc).Name())
	}
	return name
}

// IsDocker returns true if the process is running in a docker container.
func IsDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}
