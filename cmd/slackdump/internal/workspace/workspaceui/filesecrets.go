package workspaceui

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/joho/godotenv"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v3/internal/structures"
)

func fileWithSecrets(ctx context.Context, mgr manager) error {
	var filename string

	form := huh.NewForm(huh.NewGroup(
		huh.NewFilePicker().
			Title("Choose a file with secrets").
			Description("The one with SLACK_TOKEN and SLACK_COOKIE environment variables").
			ShowHidden(true).
			ShowSize(true).
			ShowPermissions(true).
			Value(&filename).
			Validate(validateSecrets),
	)).WithTheme(ui.HuhTheme()).WithHeight(10)
	if err := form.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
	}
	tok, cookie, err := parseSecretsTxt(filename)
	if err != nil {
		return err
	}
	prov, err := auth.NewValueAuth(tok, cookie)
	if err != nil {
		return err
	}
	wsp, err := createAndSelect(ctx, mgr, prov)
	if err != nil {
		return err
	}

	return success(ctx, wsp)
}

func validateSecrets(filename string) error {
	_, _, err := parseSecretsTxt(filename)
	return err
}

func parseSecretsTxt(filename string) (string, string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	secrets, err := godotenv.Parse(f)
	if err != nil {
		return "", "", errors.New("not a secrets file")
	}
	token, ok := secrets["SLACK_TOKEN"]
	if !ok {
		return "", "", errors.New("no SLACK_TOKEN found")
	}
	if err := structures.ValidateToken(token); err != nil {
		return "", "", err
	}
	if !strings.HasPrefix(token, "xoxc-") {
		return token, "", nil
	}
	cook, ok := secrets["SLACK_COOKIE"]
	if !ok {
		return "", "", errors.New("no SLACK_COOKIE found")
	}
	if !strings.HasPrefix(cook, "xoxd-") {
		return "", "", errors.New("invalid cookie")
	}
	return token, cook, nil
}
