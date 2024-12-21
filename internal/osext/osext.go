// Package osext provides some helpful os functions.
package osext

import (
	"fmt"
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
