package dirproc

import (
	"context"
	"fmt"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/processor"
)

// Users is a users processor, writes users into the users.json.gz file.
type Users struct {
	*baseproc
	cb func([]slack.User) error
}

var _ processor.Users = &Users{}

type UserOption func(*Users)

// WithUsers sets the users callback.
func WithUsers(cb func([]slack.User) error) UserOption {
	return func(u *Users) {
		u.cb = cb
	}
}

// NewUsers creates a new Users processor.
func NewUsers(cd *chunk.Directory, opt ...UserOption) (*Users, error) {
	p, err := newBaseProc(cd, "users")
	if err != nil {
		return nil, err
	}
	u := &Users{baseproc: p}
	for _, o := range opt {
		o(u)
	}
	return u, nil
}

func (u *Users) Users(ctx context.Context, users []slack.User) error {
	if err := u.baseproc.Users(ctx, users); err != nil {
		return err
	}
	if u.cb != nil {
		if err := u.cb(users); err != nil {
			return fmt.Errorf("users callback: %w", err)
		}
	}
	return nil
}
