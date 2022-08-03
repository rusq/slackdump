package auth_ui

import (
	"io"

	"github.com/AlecAivazis/survey/v2"
)

type Survey struct{}

func (*Survey) RequestWorkspace(w io.Writer) (string, error) {
	return surveyInput(
		"Enter Slack Workspace Name: ",
		"HELP:\n1. Enter the slack workspace name or paste the URL of your slack workspace.\n"+
			"2. Browser will open, login as usual.\n"+
			"3. Browser will close and slackdump will be authenticated.\n\n"+
			"This must be done only once.  The credentials are saved in an encrypted\nfile, and can be used only on this device.",
		survey.Required,
	)
}

func (*Survey) Stop() {}

func surveyInput(msg, help string, validator survey.Validator) (string, error) {
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
