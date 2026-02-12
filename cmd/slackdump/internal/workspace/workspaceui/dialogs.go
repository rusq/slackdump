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
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
)

func askRetry(ctx context.Context, name string, err error) (retry bool) {
	msg := fmt.Sprintf("The following error occurred: %s", err)
	if name != "" {
		msg = fmt.Sprintf("Error creating workspace %q: %s", name, err)
	}

	if err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title("Error Creating Workspace").
			Description(msg).
			Value(&retry).Affirmative("Retry").Negative("Cancel"),
	)).WithTheme(ui.HuhTheme()).RunWithContext(ctx); err != nil {
		return false
	}
	return retry
}

func success(ctx context.Context, workspace string) error {
	return huh.NewForm(huh.NewGroup(
		huh.NewNote().Title("Great Success!").
			Description(fmt.Sprintf("Workspace %q was added and selected.\n\n", workspace)).
			Next(true).
			NextLabel("Exit"),
	)).WithTheme(ui.HuhTheme()).RunWithContext(ctx)
}
