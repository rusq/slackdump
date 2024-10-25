// Package dumpui provides a universal wizard for running dump-family commands.
package dumpui

import (
	"context"

	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
)

type Wizard struct {
	Title       string
	Particulars string
	Cmd         *base.Command
}

const (
	actRun    = "run"
	actConfig = "config"
	actExit   = "exit"
)

func (w *Wizard) Run(ctx context.Context) error {
	var (
		action string = actRun
	)

	menu := func() *huh.Form {
		return huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(w.Title).
					Options(
						huh.NewOption("Run "+w.Particulars, actRun),
						huh.NewOption("Configuration...", actConfig),
						huh.NewOption(ui.MenuSeparator, ""),
						huh.NewOption("<< Exit to Main Menu", actExit),
					).Value(&action),
			),
		).WithTheme(ui.HuhTheme).WithAccessible(cfg.AccessibleMode)
	}

LOOP:
	for {
		if err := menu().RunWithContext(ctx); err != nil {
			return err
		}
		switch action {
		case actRun:
			if err := w.Cmd.Run(ctx, w.Cmd, nil); err != nil {
				return err
			}
		case actConfig:
			if err := cfgui.Show(ctx); err != nil {
				return err
			}
		case actExit:
			break LOOP
		}
	}

	return nil
}
