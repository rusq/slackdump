package dirproc

import "github.com/rusq/slackdump/v3/internal/chunk"

// Workspace is a processor that writes the workspace information into the
// workspace file.
type Workspace struct {
	*dirproc
}

// NewWorkspace creates a new workspace processor.
func NewWorkspace(cd *chunk.Directory) (*Workspace, error) {
	p, err := newDirProc(cd, chunk.FWorkspace)
	if err != nil {
		return nil, err
	}
	return &Workspace{dirproc: p}, nil
}
