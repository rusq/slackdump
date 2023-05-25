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

// Namer is an interface that allows us to get the name of the file.
type Namer interface {
	// Name should return the name of the file.  *os.File implements this
	// interface.
	Name() string
}
