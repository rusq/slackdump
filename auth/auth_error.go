package auth

import (
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v3/internal/structures"
)

// Error is the error returned by New, the underlying Err contains
// an API error returned by slack.AuthTest call.
type Error struct {
	Err error
	Msg string
}

func (ae *Error) Error() string {
	var msg = ae.Msg
	if msg == "" {
		msg = ae.Err.Error()
	}
	return fmt.Sprintf("authentication error: %s", msg)
}

func (ae *Error) Unwrap() error {
	return ae.Err
}

func (ae *Error) Is(target error) bool {
	return target == ae.Err
}

func IsInvalidAuthErr(err error) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	return structures.IsSlackResponseError(e.Err, "invalid_auth")
}
