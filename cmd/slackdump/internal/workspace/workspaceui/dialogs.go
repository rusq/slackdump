package workspaceui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
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
