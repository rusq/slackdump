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

	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
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
