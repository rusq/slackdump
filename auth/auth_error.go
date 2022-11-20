package auth

import "fmt"

// Error is the error returned by New, the underlying Err contains
// an API error returned by slack.AuthTest call.
type Error struct {
	Err error
}

func (ae *Error) Error() string {
	return fmt.Sprintf("failed to authenticate: %s", ae.Err)
}

func (ae *Error) Unwrap() error {
	return ae.Err
}

func (ae *Error) Is(target error) bool {
	return target == ae.Err
}
