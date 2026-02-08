// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package auth

import (
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v4/internal/structures"
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
