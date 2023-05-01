package dirproc

type Workspace struct {
	*baseproc
}

// NewWorkspace creates a new workspace processor.
func NewWorkspace(dir string) (*Workspace, error) {
	p, err := newBaseProc(dir, "workspace")
	if err != nil {
		return nil, err
	}
	return &Workspace{baseproc: p}, nil
}
