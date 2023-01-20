package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
)

const (
	dateHint = "YYYY-MM-DD"
	timeHint = "HH:MM:SS"
)

// Time asks the user to enter a date and time.
func Time(msg string, opt ...Option) (time.Time, error) {
	var opts = defaultOpts().apply(opt...)
	// q returns a survey.Question for the given entity (date or time).
	q := func(msg, entity, hint, layout string) *survey.Question {
		return &survey.Question{
			Name: entity,
			Prompt: &survey.Input{
				Message: fmt.Sprintf("%s %s (%s):", msg, strings.ToLower(entity), hint),
			},
			Validate: survey.ComposeValidators(
				survey.Required,
				func(ans interface{}) error {
					_, err := time.Parse(layout, ans.(string))
					if err != nil {
						return fmt.Errorf("invalid input, expected %s format: %s", strings.ToLower(entity), hint)
					}
					return nil
				},
			),
		}
	}

	qs := []*survey.Question{
		q(msg, "Date", dateHint, "2006-01-02"),
		q(msg, "Time", timeHint, "15:04:05"),
	}

	var p struct {
		Date string
		Time string
	}
	if err := survey.Ask(qs, &p, opts.surveyOpts()...); err != nil {
		return time.Time{}, err
	}
	res, err := time.Parse("2006-01-02 15:04:05", p.Date+" "+p.Time)
	if err != nil {
		return time.Time{}, err
	}
	return res, nil
}
