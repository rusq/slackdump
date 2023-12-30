package auth_ui

import (
	"io"

	"github.com/charmbracelet/huh"
)

type Huh struct {
	theme huh.Theme
}

func (*Huh) RequestWorkspace(w io.Writer) (string, error) {
	var workspace string
	huh.NewInput().
		Title("Enter Slack workspace name").
		Value(&workspace).
		Validate(valRequired).
		Description("The workspace name is the part of the URL that comes before `.slack.com' in\nhttps://<workspace>.slack.com/.  Both workspace name or URL are acceptable.").
		Run()
	return Sanitize(workspace)
}

func (*Huh) Stop() {}

func (*Huh) RequestEmail(w io.Writer) (string, error) {
	var email string
	huh.NewInput().Title("Enter Slack login email").
		Value(&email).
		Description("The email that you use to login to Slack.").
		Validate(valAND(valEmail, valRequired)).
		Run()
	return email, nil
}

func (*Huh) RequestPassword(w io.Writer, account string) (string, error) {
	var password string
	huh.NewInput().Title("Enter password for " + account).
		Value(&password).
		Password(true).
		Description("This is your Slack password, it will not be saved.").
		Validate(valRequired).
		Run()
	return password, nil
}

func (*Huh) RequestLoginType(w io.Writer) (int, error) {
	var loginType int
	err := huh.NewSelect[int]().Title("Select login type").
		Options(
			huh.NewOption("Email", LoginEmail),
			huh.NewOption("Google", LoginSSO),
			huh.NewOption("Apple", LoginSSO),
			huh.NewOption("Login with Single-Sign-On (SSO)", LoginSSO),
			huh.NewOption("Other", LoginSSO),
		).
		Value(&loginType).
		Description("If you are not sure, select 'Other'.").
		Run()
	return loginType, err
}
