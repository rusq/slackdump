package ui

import "github.com/AlecAivazis/survey/v2"

func Confirm(msg string, defavlt bool) (bool, error) {
	q := &survey.Confirm{
		Message: msg,
		Default: defavlt,
	}

	var b bool
	if err := survey.AskOne(q, &b); err != nil {
		return false, err
	}
	return b, nil
}
