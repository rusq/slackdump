package dirproc

import "github.com/rusq/slackdump/v3/internal/chunk"

// Workspace is a processor that writes the workspace information into the
// workspace file.
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
