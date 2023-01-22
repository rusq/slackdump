package ui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
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
func Time(msg string, opt ...Option) (time.Time, error) {
	var opts = defaultOpts().apply(opt...)
	// q returns a survey.Question for the given entity (date or time).
	q := func(msg, entity, hint, layout string, required bool) *survey.Question {
		return &survey.Question{
			Name: entity,
			Prompt: &survey.Input{
				Message: fmt.Sprintf("%s %s (%s):", msg, strings.ToLower(entity), hint),
			},
			Validate: survey.ComposeValidators(
				func(ans interface{}) error {
					s := ans.(string)
					if !required && s == "" {
						return nil
					}
					_, err := time.Parse(layout, ans.(string))
					if err != nil {
						return fmt.Errorf("invalid input, expected %s format: %s", strings.ToLower(entity), hint)
					}
					return nil
				},
			),
		}
	}

	var p struct {
		Date string
		Time string
	}

	// First, ask for date.  Date is optional.  If date is not given, we
	// shall not ask for time, and will return EmptyOptionalInput.
	if err := survey.Ask(
		[]*survey.Question{q(msg, "Date", dateHint, "2006-01-02", false)},
		&p,
		opts.surveyOpts()...,
	); err != nil {
		return time.Time{}, err
	}
	if p.Date == "" {
		return time.Time{}, ErrEmptyOptionalInput
	}
	// if date is given, ask for time.  Time is required.
	if err := survey.Ask(
		[]*survey.Question{q(msg, "Time", timeHint, "15:04:05", true)},
		&p,
		opts.surveyOpts()...,
	); err != nil {
		return time.Time{}, err
	}

	res, err := time.Parse("2006-01-02 15:04:05", p.Date+" "+p.Time)
	if err != nil {
		return time.Time{}, err
	}
	return res, nil
}
