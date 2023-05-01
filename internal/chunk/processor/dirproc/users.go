package dirproc

import "github.com/rusq/slackdump/v2/internal/chunk"

// Users is a users processor.
type Users struct {
	*baseproc
}

// NewUsers creates a new Users processor.
func NewUsers(cd *chunk.Directory) (*Users, error) {
	p, err := newBaseProc(cd, "users")
	if err != nil {
		return nil, err
	}
	return &Users{baseproc: p}, nil
}
