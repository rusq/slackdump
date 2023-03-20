package expproc

// Users is a users processor.
type Users struct {
	*baseproc
}

// NewUsers creates a new Users processor.
func NewUsers(dir string) (*Users, error) {
	p, err := newBaseProc(dir, "users.json")
	if err != nil {
		return nil, err
	}
	return &Users{baseproc: p}, nil
}
