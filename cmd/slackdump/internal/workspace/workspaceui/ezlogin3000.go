package workspaceui

import (
	"context"
	"errors"

	"github.com/rusq/slackdump/v3/auth"

	"github.com/rusq/slackdump/v3/auth/browser"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"

	"github.com/charmbracelet/huh"
)

func ezLogin3000(ctx context.Context, mgr manager) error {
	var (
		legacy bool
	)
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Use legacy EZ-Login?").
			Description("Do you want to use the legacy login?").
			Value(&legacy),
	)).WithTheme(ui.HuhTheme()).WithKeyMap(ui.DefaultHuhKeymap)
	if err := form.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		return err
	}
	if legacy {
		return playwrightLogin(ctx, mgr)
	}
	return rodLogin(ctx, mgr)

}

func playwrightLogin(ctx context.Context, mgr manager) error {
	var brws = browser.Bchromium
	formBrowser := huh.NewForm(huh.NewGroup(
		huh.NewSelect[browser.Browser]().
			Options(
				huh.NewOption("Chromium", browser.Bchromium),
				huh.NewOption("Firefox", browser.Bfirefox),
			).
			Value(&brws),
	)).WithTheme(ui.HuhTheme()).WithKeyMap(ui.DefaultHuhKeymap)
	if err := formBrowser.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		return err
	}
	prov, err := auth.NewBrowserAuth(ctx, auth.BrowserWithBrowser(brws))
	if err != nil {
		return err
	}

	name, err := createAndSelect(ctx, mgr, prov)
	if err != nil {
		return err
	}
	return success(ctx, name)
}

func rodLogin(ctx context.Context, mgr manager) error {
	prov, err := auth.NewRODAuth(ctx)
	if err != nil {
		return err
	}
	name, err := createAndSelect(ctx, mgr, prov)
	if err != nil {
		return err
	}
	return success(ctx, name)
}
