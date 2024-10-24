package auth_ui

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/charmbracelet/huh"
)

// Huh is the Auth UI that uses the huh library to provide a terminal UI.
type Huh struct{}

func (h *Huh) RequestWorkspace(w io.Writer) (string, error) {
	var workspace string
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Enter Slack workspace name").
			Value(&workspace).
			Validate(valRequired).
			Description("The workspace name is the part of the URL that comes before `.slack.com' in\nhttps://<workspace>.slack.com/.  Both workspace name or URL are acceptable."),
	)).Run()
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
				Placeholder("you@work.com").
				Description(fmt.Sprintf("This is the email that you log into %s with.", workspace)).
				Validate(valAND(valEmail, valRequired)),
			huh.NewInput().
				Title("Password").Value(&passwd).
				Placeholder("your slack password").
				Validate(valRequired).EchoMode(huh.EchoModePassword),
		),
	)
	err = f.Run()
	return
}

func (*Huh) RequestLoginType(w io.Writer) (LoginType, error) {
	var loginType LoginType
	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[LoginType]().Title("Select login type").
			Options(
				huh.NewOption("Email (manual)", LInteractive),
				huh.NewOption("Email (automatic)", LHeadless),
				huh.NewOption("Google", LGoogleAuth),
				huh.NewOption("Apple", LInteractive),
				huh.NewOption("Login with Single-Sign-On (SSO)", LInteractive),
				huh.NewOption("Other/Manual", LInteractive),
				huh.NewOption("", LoginType(-1)),
				huh.NewOption("Cancel", LCancel),
			).
			Value(&loginType).
			Validate(valSepEaster()).
			DescriptionFunc(func() string {
				switch loginType {
				case LInteractive:
					return "Clean browser will open on a Slack Login page."
				case LHeadless:
					return "You will be prompted to enter your email and password, login is automated."
				case LGoogleAuth:
					return "System browser will open on a Slack Login page."
				case LCancel:
					return "Cancel the login process."
				default:
					return ""
				}
			}, &loginType),
	)).Run()
	return loginType, err
}

// ConfirmationCode asks the user to input the confirmation code, does some
// validation on it and returns it as an int.
func (*Huh) ConfirmationCode(email string) (int, error) {
	var strCode string
	q := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			CharLimit(6).
			Placeholder("00000").
			Title(fmt.Sprintf("Enter confirmation code sent to %s", email)).
			Description("Slack did not recognise the browser, and sent a confirmation code.  Please enter the confirmation code below.").
			Value(&strCode).
			Validate(valSixDigits),
	))
	if err := q.Run(); err != nil {
		return 0, err
	}
	code, err := strconv.Atoi(strCode)
	if err != nil {
		return 0, err
	}
	return code, nil
}

var numChlgRE = regexp.MustCompile(`^\d{6}$`)

func valSixDigits(s string) error {
	if !numChlgRE.MatchString(s) {
		return errors.New("confirmation code must be a sequence of six digits")
	}
	return nil
}
