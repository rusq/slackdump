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
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v4/internal/structures"
)

const sampleToken = "xoxc-610187951300-604451271234-3473161557912-4c426dd426a45208707725b710302b32dda0ab002b80ccd8c4c8ac9971a11558"

func prgTokenCookie(ctx context.Context, mgr manager) error {
	var (
		token     string
		cookie    string
		workspace string
		confirmed bool
	)

	for !confirmed {
		f := huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Token").
				Description("Token value").
				Placeholder(sampleToken).
				Value(&token).
				Validate(structures.ValidateToken),
			huh.NewInput().Title("Cookie").
				Description("Session cookie").
				Placeholder("xoxd-...").
				Value(&cookie),
			huh.NewConfirm().Title("Confirm creation of workspace?").
				Description("Once confirmed this will check the credentials for validity, detect the workspace \nand create a new workspace with the provided token and cookie").
				Value(&confirmed).
				Validate(makeValidator(ctx, &token, &cookie, auth.NewValueAuth)),
		)).WithTheme(ui.HuhTheme()).WithKeyMap(ui.DefaultHuhKeymap)
		if err := f.RunWithContext(ctx); err != nil {
			return err
		}
		if !confirmed {
			return nil
		}

		prov, err := auth.NewValueAuth(token, cookie)
		if err != nil {
			return err
		}
		name, err := mgr.CreateAndSelect(ctx, prov)
		if err != nil {
			confirmed = false
			retry := askRetry(ctx, name, err)
			if !retry {
				return nil
			}
		} else {
			workspace = name
			break
		}
	}

	return success(ctx, workspace)
}

// makeValidator creates a validator function that uses the newProvFn to
// create a new provider and test it.  newProvFn should be a function that
// creates a new provider from a token and a value, where value is either a
// cookie or a file with cookies.
func makeValidator[P auth.Provider](ctx context.Context, token *string, val *string, newProvFn func(string, string) (P, error)) func(bool) error {
	return func(b bool) error {
		if !b {
			return nil
		}
		p, err := newProvFn(*token, *val)
		if err != nil {
			return err
		}
		_, err = p.Test(ctx)
		if err != nil {
			return err
		}
		return nil
	}
}

func prgTokenCookieFile(ctx context.Context, mgr manager) error {
	var (
		token      string
		cookiefile string
		workspace  string
		confirmed  bool
	)
	for !confirmed {
		f := huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Token").
				Description("Token value").
				Placeholder(sampleToken).
				Value(&token).
				Validate(structures.ValidateToken),
			huh.NewFilePicker().Title("Cookie File").
				Description("Select a cookies.txt file in Mozilla Format").AllowedTypes([]string{"txt"}).
				FileAllowed(true).
				ShowSize(true).
				ShowPermissions(true).
				Value(&cookiefile),
			huh.NewConfirm().Title("Is this correct?").
				Description("Once confirmed this will create a new workspace with the provided token and cookie").
				Value(&confirmed).
				Validate(makeValidator(ctx, &token, &cookiefile, auth.NewCookieFileAuth)),
		)).WithTheme(ui.HuhTheme()).WithKeyMap(ui.DefaultHuhKeymap)
		if err := f.Run(); err != nil {
			return err
		}

		prov, err := auth.NewValueAuth(token, cookiefile)
		if err != nil {
			return err
		}
		name, err := mgr.CreateAndSelect(ctx, prov)
		if err != nil {
			confirmed = false
			retry := askRetry(ctx, name, err)
			if !retry {
				return nil
			}
		} else {
			workspace = name
			break
		}
	}

	return success(ctx, workspace)
}

func prgCookieOnly(ctx context.Context, mgr manager) error {
	var (
		wspname   string
		cookie    string
		confirmed bool
	)

	newCookieProvFn := func(wsp, cookie string) (auth.ValueAuth, error) {
		return auth.NewCookieOnlyAuth(ctx, wsp, cookie)
	}

	for !confirmed {
		f := huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Workspace").
				Description("Workspace Name (just the name, not the URL)").
				Placeholder("my-team-workspace").
				Value(&wspname).
				Validate(validateWspName),
			huh.NewInput().Title("Cookie").
				Description("Session cookie").
				Placeholder("xoxd-...").
				Value(&cookie),
			huh.NewConfirm().Title("Confirm creation of workspace?").
				Description("Once confirmed this will check the credentials for validity, fetch the token\nand create a new workspace").
				Value(&confirmed).
				Validate(makeValidator(ctx, &wspname, &cookie, newCookieProvFn)),
		)).WithTheme(ui.HuhTheme()).WithKeyMap(ui.DefaultHuhKeymap)
		if err := f.RunWithContext(ctx); err != nil {
			return err
		}
		if !confirmed {
			return nil
		}

		prov, err := auth.NewCookieOnlyAuth(ctx, wspname, cookie)
		if err != nil {
			return err
		}
		name, err := mgr.CreateAndSelect(ctx, prov)
		if err != nil {
			confirmed = false
			retry := askRetry(ctx, name, err)
			if !retry {
				return nil
			}
		} else {
			break
		}
	}

	return success(ctx, wspname)
}

func validateWspName(s string) error {
	if strings.Contains(s, "://") || strings.Contains(s, "slack.com") {
		return errors.New("workspace name, not URL, i.e. for https://my-team.slack.com, enter my-team")
	}
	return nil
}
