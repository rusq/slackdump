package archive

import (
	"context"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/wizard"
)

func archiveWizard(ctx context.Context, cmd *base.Command, args []string) error {
	var (
		action string = "run"
	)

	menu := func() *huh.Form {
		return huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Key("selection").
					Title("Archive Slackdump workspace").
					Options(
						huh.NewOption("Run!", "run"),
						huh.NewOption("Configure", "config"),
						huh.NewOption("Show Config", "show"),
						huh.NewOption(strings.Repeat("-", 10), ""),
						huh.NewOption("Exit archive wizard", "exit"),
					).Value(&action),
			),
		).WithTheme(cfg.Theme).WithAccessible(cfg.AccessibleMode)
	}

LOOP:
	for {
		if err := menu().RunWithContext(ctx); err != nil {
			return err
		}
		switch action {
		case "exit":
			break LOOP
		case "config":
			if err := wizard.Config(ctx); err != nil {
				return err
			}
		case "show":
			if err := cfgui.Show(ctx); err != nil {
				return err
			}
		case "run":
			//TODO: implement archive
		}
	}

	return nil
}
