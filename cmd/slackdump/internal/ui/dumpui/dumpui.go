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

// Wizard is a universal wizard for running dump-family commands.
type Wizard struct {
	// Title is the title of the command.
	Title string
	// Name is the name of the command.
	Name string
	// LocalConfig should return a configuration for the command.
	LocalConfig func() cfgui.Configuration
	// ArgsFn should return a slice of arguments to pass to the command.
	ArgsFn func() []string
	// ValidateParamsFn should return true if the parameters are OK.
	ValidateParamsFn func() error
	// Cmd is the command to run.
	Cmd *base.Command
}

const (
	actRun          = "run"
	actGlobalConfig = "config"
	actLocalConfig  = "localconfig"
	actExit         = "exit"
)

var description = map[string]string{
	actRun:          "Run the command",
	actGlobalConfig: "Set global configuration options",
	actLocalConfig:  "Set command specific configuration options",
	actExit:         "Exit to main menu",
}

func (w *Wizard) Run(ctx context.Context) error {
	var (
		action string = actRun
	)

	menu := func(opts ...huh.Option[string]) *huh.Form {
		return huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(w.Title).
					Options(
						opts...,
					).Value(&action).
					DescriptionFunc(func() string { return description[action] }, &action),
			),
		).WithTheme(ui.HuhTheme).WithAccessible(cfg.AccessibleMode)
	}

LOOP:
	for {
		var opts []huh.Option[string]
		if w.ValidateParamsFn != nil && w.LocalConfig != nil {
			if err := w.ValidateParamsFn(); err == nil {
				opts = append(opts, huh.NewOption("Run "+w.Name, actRun))
			}
			action = actLocalConfig
		} else {
			opts = append(opts, huh.NewOption("Run "+w.Name, actRun))
		}
		// local config
		if w.LocalConfig != nil {
			opts = append(opts, huh.NewOption(w.Name+" Configuration...", actLocalConfig))
		}
		// final options
		opts = append(opts,
			huh.NewOption("Global Configuration...", actGlobalConfig),
			huh.NewOption(ui.MenuSeparator, ""),
			huh.NewOption("<< Exit to Main Menu", actExit),
		)

		if err := menu(opts...).RunWithContext(ctx); err != nil {
			return err
		}
		switch action {
		case actRun:
			if w.ValidateParamsFn != nil {
				if err := w.ValidateParamsFn(); err != nil {
					continue
				}
			}
			var args []string
			if w.ArgsFn != nil {
				args = w.ArgsFn()
			}
			if err := w.Cmd.Run(ctx, w.Cmd, args); err != nil {
				return err
			}
		case actGlobalConfig:
			if err := cfgui.Global(ctx); err != nil {
				return err
			}
		case actLocalConfig:
			if err := cfgui.Local(ctx, w.LocalConfig); err != nil {
				return err
			}
		case actExit:
			break LOOP
		}
	}

	return nil
}
