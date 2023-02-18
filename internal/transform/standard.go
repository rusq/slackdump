package transform

import "github.com/rusq/slackdump/v2/fsadapter"

type Standard struct {
	fs fsadapter.FS
}

func NewStandard(fs fsadapter.FS, fsdir string) *Standard {
	return &Standard{fs: fs}
}

func (s *Standard) Do() error {
	return nil
}
