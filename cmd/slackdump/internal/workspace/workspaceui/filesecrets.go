package workspaceui

import (
	"context"
	"errors"

	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
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
	)).WithTheme(ui.HuhTheme()).WithHeight(10).WithKeyMap(ui.DefaultHuhKeymap)
	if err := form.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
	}

	tok, cookie, err := auth.ParseDotEnv(filename)
	if err != nil {
		return err
	}
	prov, err := auth.NewValueAuth(tok, cookie)
	if err != nil {
		return err
	}
	name, err := mgr.CreateAndSelect(ctx, prov)
	if err != nil {
		return err
	}

	return success(ctx, name)
}

func validateSecrets(filename string) error {
	_, _, err := auth.ParseDotEnv(filename)
	return err
}
