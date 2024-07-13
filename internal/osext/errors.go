package osext

import (
	"errors"
	"io/fs"
)

// IsPathError reports whether the error is an fs.PathError.
func IsPathError(err error) bool {
	var pathError *fs.PathError
	return errors.As(err, &pathError)
}
