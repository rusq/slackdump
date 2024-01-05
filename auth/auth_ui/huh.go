package auth_ui

import (
	"errors"
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
			huh.NewOption("------", -1),
			huh.NewOption("Cancel", LoginCancel),
		).
		Value(&loginType).
		Validate(valSepEaster()).
		Description("If you are not sure, select 'Other'.").
		Run()
	return loginType, err
}

func valSepEaster() func(v int) error {
	var phrases = []string{
		"This is a separator, it does nothing",
		"Seriously, it does nothing",
		"Stop clicking on it",
		"Stop it",
		"Stop",
		"Why are you so persistent?",
		"Fine, you win",
		"Here's a cookie: 🍪",
		"🍪",
		"🍪",
		"Don't be greedy, you already had three.",
		"Ok, here's another one: 🍪",
		"Nothing will happen if you click on it again",
		"",
		"",
		"",
		"You must have a lot of time on your hands",
		"Or maybe you're just bored",
		"Or maybe you're just procrastinating",
		"Or maybe you're just trying to get a cookie",
		"These are virtual cookies, you can't eat them, but here's another one: 🍪",
		"🍪",
		"You have reached the end of this joke, it will now repeat",
		"Seriously...",
		"Ah, shit, here we go again",
	}
	var i int
	return func(v int) error {
		if v == -1 {
			// separator selected
			msg := phrases[i]
			i = (i + 1) % len(phrases)
			return errors.New(msg)
		}
		return nil
	}
}
