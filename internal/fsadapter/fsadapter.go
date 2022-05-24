package fsadapter

import "io"

// FileCreator is interface for saving files to underlying filesystem.
type FileCreator interface {
	Create(string) (io.WriteCloser, error)
}

type FileCreateRemover interface {
	FileCreator
	RemoveAll(string) error
}
