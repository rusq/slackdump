// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package workspaceui

import (
	"context"
	"errors"

	"github.com/rusq/slackdump/v4/auth"

	"github.com/rusq/slackdump/v4/auth/browser"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/workspace/wspcfg"

	"github.com/charmbracelet/huh"
)

func brwsLogin() func(ctx context.Context, mgr manager) error {
	return func(ctx context.Context, mgr manager) error {
		var err error
		if wspcfg.LegacyBrowser {
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
	brws := browser.Bchromium
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
	prov, err := auth.NewPlaywrightAuth(ctx, auth.BrowserWithBrowser(brws), auth.BrowserWithTimeout(wspcfg.LoginTimeout))
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
	prov, err := auth.NewRODAuth(ctx, auth.BrowserWithTimeout(wspcfg.LoginTimeout), auth.RODWithRODHeadlessTimeout(wspcfg.HeadlessTimeout), auth.RODWithUserAgent(wspcfg.RODUserAgent))
	if err != nil {
		return err
	}
	name, err := mgr.CreateAndSelect(ctx, prov)
	if err != nil {
		return err
	}
	return success(ctx, name)
}
