// Package osext provides some helpful os functions.
package osext

import "fmt"

type Error struct {
	File string
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Err, e.File)
}
