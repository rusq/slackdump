package structures

import (
	"errors"
	"strings"

	"github.com/rusq/slack"
)

// IsSlackResponseError resturns true if the following conditions are met:
// - error is of [slack.SlackErrorResponse] type; AND
// - e.Err field equal to the string s.
// otherwise, returns false.
func IsSlackResponseError(e error, s string) bool {
	var se slack.SlackErrorResponse
	return errors.As(e, &se) && strings.EqualFold(se.Err, s)
}
