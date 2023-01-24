package ui

import "github.com/AlecAivazis/survey/v2"

func Confirm(msg string, defavlt bool, opt ...Option) (bool, error) {
	var opts = defaultOpts().apply(opt...)

	q := &survey.Confirm{
		Message: msg,
		Help:    opts.help,
		Default: defavlt,
	}

	var b bool
	if err := survey.AskOne(q, &b, opts.surveyOpts()...); err != nil {
		return false, err
	}
	return b, nil
}
