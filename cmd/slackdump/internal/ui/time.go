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

package ui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

const (
	dateHint = "YYYY-MM-DD"
	timeHint = "HH:MM:SS"
)

// ErrEmptyOptionalInput is returned when an optional input is empty.
var ErrEmptyOptionalInput = errors.New("empty input in optional field")

// Time asks the user to enter a date and time.  For simplicity, the date and
// time are entered in two separate prompts.  The date is optional, and if
// it is not given, the function terminates returning ErrEmptyOptionalInput.
// If the date is entered and is valid (checked with validators, you don't have
// to worry), the function will ask for time, which is then required.
func Time(msg string, _ ...Option) (time.Time, error) {
	// q returns a survey.Question for the given entity (date or time).
	q := func(msg, entity, hint, layout string, required bool) *huh.Input {
		return huh.NewInput().
			Title(fmt.Sprintf("%s %s (%s):", msg, strings.ToLower(entity), hint)).
			Validate(func(s string) error {
				if !required && s == "" {
					return nil
				}
				_, err := time.Parse(layout, s)
				if err != nil {
					return fmt.Errorf("invalid input, expected %s format: %s", strings.ToLower(entity), hint)
				}
				return nil
			})
	}

	var p struct {
		Date string
		Time string
	}

	// First, ask for date.  Date is optional.  If date is not given, we
	// shall not ask for time, and will return EmptyOptionalInput.
	if err := q(msg, "Date", dateHint, "2006-01-02", false).Value(&p.Date).Run(); err != nil {
		return time.Time{}, err
	}
	if p.Date == "" {
		return time.Time{}, ErrEmptyOptionalInput
	}
	// if date is given, ask for time.  Time is required.
	if err := q(msg, "Time", timeHint, "15:04:05", true).Value(&p.Time).Run(); err != nil {
		return time.Time{}, err
	}

	res, err := time.Parse("2006-01-02 15:04:05", p.Date+" "+p.Time)
	if err != nil {
		return time.Time{}, err
	}
	return res, nil
}
