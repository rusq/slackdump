package ui

import "github.com/AlecAivazis/survey/v2"

// Input shows a text input field with a custom validator.
func Input(msg, help string, validator survey.Validator) (string, error) {
	qs := []*survey.Question{
		{
			Name:     "value",
			Validate: validator,
			Prompt: &survey.Input{
				Message: msg,
				Help:    help,
			},
		},
	}
	var m = struct {
		Value string
	}{}
	if err := survey.Ask(qs, &m); err != nil {
		return "", err
	}
	return m.Value, nil
}

// StringRequire requires user to input string.
func StringRequire(msg, help string) (string, error) {
	return Input(msg, help, survey.Required)
}

// String asks user to input string, accepts an empty input.
func String(msg, help string) (string, error) {
	return Input(msg, help, nil)
}
