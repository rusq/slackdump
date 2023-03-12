// Package osext provides some extended functionality for the os package.
package osext

import "fmt"

type Error struct {
	File string
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Err, e.File)
}
