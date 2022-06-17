package slackdump

import "fmt"

type AuthError struct {
	Err error
}

func (ae *AuthError) Error() string {
	return fmt.Sprintf("failed to authenticate: %s", ae.Err)
}

func (ae *AuthError) Unwrap() error {
	return ae.Err
}
