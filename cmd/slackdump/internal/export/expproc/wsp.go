package expproc

type Workspace struct {
	*baseproc
}

func NewWorkspace(dir string) (*Workspace, error) {
	p, err := newBaseProc(dir, "workspace")
	if err != nil {
		return nil, err
	}
	return &Workspace{baseproc: p}, nil
}
