package auth_ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/rusq/slackauth"

	"github.com/rusq/slackdump/v3/internal/structures"
)

// Huh is the Auth UI that uses the huh library to provide a terminal UI.
type Huh struct{}

var Theme = huh.ThemeBase16()

func (h *Huh) RequestWorkspace(w io.Writer) (string, error) {
	var workspace string
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Enter Slack workspace name").
			Value(&workspace).
			Validate(valWorkspace).
			Description("The workspace name is the part of the URL that comes before `.slack.com' in\nhttps://<workspace>.slack.com/.  Both workspace name or URL are acceptable."),
	)).WithTheme(Theme).WithKeyMap(keymap).Run()
	if err != nil {
		return "", err
	}
	return workspace, nil
}

func (*Huh) Stop() {}

func (*Huh) RequestCreds(ctx context.Context, w io.Writer, workspace string) (email string, passwd string, err error) {
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
	).WithTheme(Theme).WithKeyMap(keymap)
	err = f.RunWithContext(ctx)
	return
}

type methodMenuItem struct {
	MenuItem  string
	ShortDesc string
	Type      LoginType
}

func (m methodMenuItem) String() string {
	return fmt.Sprintf("%-20s - %s", m.MenuItem, m.ShortDesc)
}

var gMethods = []methodMenuItem{
	{
		"Interactive",
		"Works with most authentication schemes, except Google.",
		LInteractive,
	},
	{
		"Automatic",
		"Only suitable for email/password auth.",
		LHeadless,
	},
	{
		"User Browser",
		"Loads your user profile, works with Google Auth.",
		LUserBrowser,
	},
	{
		"QR Code",
		"Login using Sign in on Mobile QR code, works with Google Auth.",
		LMobileSignin,
	},
}

type LoginOpts struct {
	Workspace   string
	Type        LoginType
	BrowserPath string
}

var keymap = huh.NewDefaultKeyMap()

func init() {
	keymap.Quit = key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "Quit"))
}

func (*Huh) RequestLoginType(ctx context.Context, _ io.Writer, workspace string) (LoginOpts, error) {
	ret := LoginOpts{
		Workspace:   workspace,
		Type:        LInteractive,
		BrowserPath: "",
	}

	opts := make([]huh.Option[LoginType], 0, len(gMethods))
	for _, m := range gMethods {
		opts = append(opts, huh.NewOption(m.String(), m.Type))
	}
	opts = append(opts,
		huh.NewOption("------", LoginType(-1)),
		huh.NewOption("Cancel", LCancel),
	)
	var fields []huh.Field
	if workspace == "" {
		fields = append(fields, huh.NewInput().
			Title("Enter Slack workspace name").
			Value(&ret.Workspace).
			Validate(valWorkspace).
			Description("The workspace name is the part of the URL that comes before `.slack.com' in\nhttps://<workspace>.slack.com/.  Both workspace name or URL are acceptable."),
		)
	}

	fields = append(fields, huh.NewSelect[LoginType]().
		TitleFunc(func() string {
			wsp, err := structures.ExtractWorkspace(ret.Workspace)
			if err != nil {
				return "Select login type"
			}
			return fmt.Sprintf("Select login type for [%s]", wsp)
		}, &ret.Workspace).
		Options(opts...).
		Value(&ret.Type).
		Validate(valSepEaster()).
		DescriptionFunc(func() string {
			switch ret.Type {
			case LInteractive:
				return "Clean browser will open on a Slack Login page."
			case LHeadless:
				return "You will be prompted to enter your email and password, login is automated."
			case LUserBrowser:
				return "System browser will open on a Slack Login page."
			case LMobileSignin:
				return "Sign in using 'Sign in on Mobile' QR code."
			case LCancel:
				return "Cancel the login process."
			default:
				return ""
			}
		}, &ret.Type))

	form := huh.NewForm(huh.NewGroup(fields...)).WithTheme(Theme).WithKeyMap(keymap)

	if err := form.RunWithContext(ctx); err != nil {
		return ret, err
	}
	var err error
	ret.Workspace, err = structures.ExtractWorkspace(ret.Workspace)
	if err != nil {
		return ret, err
	}

	if ret.Type == LUserBrowser {
		path, err := chooseBrowser(ctx)
		if err != nil {
			return ret, err
		}
		ret.BrowserPath = path
		return ret, err
	}

	return ret, nil
}

func chooseBrowser(ctx context.Context) (string, error) {
	browsers, err := slackauth.ListBrowsers()
	if err != nil {
		return "", err
	}
	opts := make([]huh.Option[int], 0, len(browsers))
	for i, b := range browsers {
		opts = append(opts, huh.NewOption(b.Name, i))
	}

	var selection int
	err = huh.NewForm(huh.NewGroup(
		huh.NewSelect[int]().
			Title("Detected browsers on your system").
			Options(opts...).
			Value(&selection).
			DescriptionFunc(func() string {
				return browsers[selection].Path
			}, &selection),
	)).WithTheme(Theme).WithKeyMap(keymap).RunWithContext(ctx)
	if err != nil {
		return "", err
	}
	return browsers[selection].Path, nil
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
	)).WithTheme(Theme)
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

const (
	maxEncImgSz = 7000
	imgPrefix   = "data:image/png;base64,"
)

func (*Huh) RequestQR(ctx context.Context, _ io.Writer) (string, error) {
	const description = `In logged in Slack Client or Web:
  1. click the username in the upper left corner;
  2. choose 'Sign in on mobile';
  3. right-click the QR code image;
  4. choose Copy Image.`
	var imageData string
	q := huh.NewForm(huh.NewGroup(
		huh.NewText().
			CharLimit(maxEncImgSz).
			Value(&imageData).
			Validate(func(s string) error {
				if !strings.HasPrefix(s, imgPrefix) {
					return errors.New("image data must start with " + imgPrefix)
				}
				return nil
			}).
			Placeholder(imgPrefix + "...").
			Title("Paste QR code image data into this field").
			Description(""),
	))
	if err := q.Run(); err != nil {
		return "", err
	}
	return imageData, nil
}
