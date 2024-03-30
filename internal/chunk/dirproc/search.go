package dirproc

import "github.com/rusq/slackdump/v3/internal/chunk"

type Search struct {
	*baseproc
}

func NewSearch(dir *chunk.Directory) (*Search, error) {
	p, err := newBaseProc(dir, "search")
	if err != nil {
		return nil, err
	}
	return &Search{baseproc: p}, nil
}
