package transform

import "github.com/rusq/fsadapter"

type Standard struct {
	fs fsadapter.FS
}

func NewStandard(fs fsadapter.FS, fsdir string, srcDir string) *Standard {
	return &Standard{fs: fs}
}

func (s *Standard) Transform() error {
	return nil
}
