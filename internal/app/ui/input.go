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

func MustString(msg, help string) (string, error) {
	return Input(msg, help, survey.Required)
}

func String(msg, help string) (string, error) {
	return Input(msg, help, nil)
}
