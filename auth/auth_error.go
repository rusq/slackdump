package auth

import (
	"errors"
	"fmt"

	"github.com/slack-go/slack"
)

// Error is the error returned by New, the underlying Err contains
// an API error returned by slack.AuthTest call.
type Error struct {
	Err error
	Msg string
}

func (ae *Error) Error() string {
	var msg string = ae.Msg
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
	var ser slack.SlackErrorResponse
	if !errors.As(e.Err, &ser) {
		return false
	}
	return ser.Err == "invalid_auth"
}
