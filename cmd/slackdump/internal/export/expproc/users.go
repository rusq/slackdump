package expproc

type Users struct {
	*baseproc
}

func NewUsers(dir string) (*Users, error) {
	p, err := newBaseProc(dir, "users.json")
	if err != nil {
		return nil, err
	}
	return &Users{baseproc: p}, nil
}
