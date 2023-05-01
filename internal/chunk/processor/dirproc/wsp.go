package dirproc

import "github.com/rusq/slackdump/v2/internal/chunk"

type Workspace struct {
	*baseproc
}

// NewWorkspace creates a new workspace processor.
func NewWorkspace(cd *chunk.Directory) (*Workspace, error) {
	p, err := newBaseProc(cd, "workspace")
	if err != nil {
		return nil, err
	}
	return &Workspace{baseproc: p}, nil
}
