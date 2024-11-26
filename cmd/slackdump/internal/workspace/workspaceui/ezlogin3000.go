package workspaceui

import (
	"context"
	"errors"

	"github.com/rusq/slackdump/v3/auth"

	"github.com/rusq/slackdump/v3/auth/browser"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"

	"github.com/charmbracelet/huh"
)

func brwsLogin(opts *browserOptions) func(ctx context.Context, mgr manager) error {
	return func(ctx context.Context, mgr manager) error {
		var err error
		if opts.UsePlaywright {
			err = playwrightLogin(ctx, mgr)
		} else {
			err = rodLogin(ctx, mgr)
		}
		if err != nil {
			if errors.Is(err, auth.ErrCancelled) {
				return nil
			}
			return err
		}
		return nil
	}
}

func playwrightLogin(ctx context.Context, mgr manager) error {
	var brws = browser.Bchromium
	formBrowser := huh.NewForm(huh.NewGroup(
		huh.NewSelect[browser.Browser]().
			Options(
				huh.NewOption("Chromium", browser.Bchromium),
				huh.NewOption("Firefox", browser.Bfirefox),
			).
			Title("Playwright login").
			Description("Choose the browser to use for authentication").
			Value(&brws),
	)).WithTheme(ui.HuhTheme()).WithKeyMap(ui.DefaultHuhKeymap)
	if err := formBrowser.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		return err
	}
	prov, err := auth.NewPlaywrightAuth(ctx, auth.BrowserWithBrowser(brws))
	if err != nil {
		return err
	}

	name, err := mgr.CreateAndSelect(ctx, prov)
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
	name, err := mgr.CreateAndSelect(ctx, prov)
	if err != nil {
		return err
	}
	return success(ctx, name)
}
