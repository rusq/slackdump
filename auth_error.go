package slackdump

import "fmt"

// AuthError is the error returned by New, the underlying Err contains
// an API error returned by slack.AuthTest call.
type AuthError struct {
	Err error
}

func (ae *AuthError) Error() string {
	return fmt.Sprintf("failed to authenticate: %s", ae.Err)
}

func (ae *AuthError) Unwrap() error {
	return ae.Err
}

func (ae *AuthError) Is(target error) bool {
	return target == ae.Err
}
