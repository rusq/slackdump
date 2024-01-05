package auth_ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
)

type Huh struct {
	theme huh.Theme
}

func (*Huh) RequestWorkspace(w io.Writer) (string, error) {
	var workspace string
	err := huh.NewInput().
		Title("Enter Slack workspace name").
		Value(&workspace).
		Validate(valRequired).
		Description("The workspace name is the part of the URL that comes before `.slack.com' in\nhttps://<workspace>.slack.com/.  Both workspace name or URL are acceptable.").
		Run()
	if err != nil {
		return "", err
	}
	return Sanitize(workspace)
}

func (*Huh) Stop() {}

func (*Huh) RequestCreds(w io.Writer, workspace string) (email string, passwd string, err error) {
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("You Slack Login Email").Value(&email).
				Description(fmt.Sprintf("This is the email that you log into %s with.", workspace)).
				Validate(valAND(valEmail, valRequired)),
			huh.NewInput().
				Title("Password").Value(&passwd).
				Validate(valRequired).Password(true),
		),
	)
	err = f.Run()
	return
}

func (*Huh) RequestLoginType(w io.Writer) (int, error) {
	var loginType int
	err := huh.NewSelect[int]().Title("Select login type").
		Options(
			huh.NewOption("Email", LoginEmail),
			huh.NewOption("Google", LoginSSO),
			huh.NewOption("Apple", LoginSSO),
			huh.NewOption("Login with Single-Sign-On (SSO)", LoginSSO),
			huh.NewOption("Other/Manual", LoginSSO),
			huh.NewOption("------", LoginCancel),
			huh.NewOption("Cancel", LoginCancel),
		).
		Value(&loginType).
		Description("If you are not sure, select 'Other'.").
		Run()
	return loginType, err
}
