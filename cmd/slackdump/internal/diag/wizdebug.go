package diag

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
)

var CmdWizDebug = &base.Command{
	UsageLine:  "slackdump tools wizdebug",
	Short:      "run the wizard debug command",
	Run:        runWizDebug,
	PrintFlags: true,
}

type wdWhat int

const (
	wdExit wdWhat = iota
	wdDumpUI
	wdConfigUI
)

func runWizDebug(ctx context.Context, cmd *base.Command, args []string) error {
	var action wdWhat
	for {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[wdWhat]().Options(
					huh.NewOption("Dump UI", wdDumpUI),
					huh.NewOption("Global Config UI", wdConfigUI),
				).Value(&action),
			).WithHeight(10),
		)

		if err := form.RunWithContext(ctx); err != nil {
			return err
		}
		switch action {
		case wdDumpUI:
			if err := debugDumpUI(ctx); err != nil {
				return err
			}
		case wdConfigUI:
			if err := debugConfigUI(ctx); err != nil {
				return err
			}
		case wdExit:
			return nil
		}
	}
}

func debugDumpUI(ctx context.Context) error {
	menu := []dumpui.MenuItem{
		{
			ID:   "run",
			Name: "Run",
			Help: "Run the command",
		},
		{
			Name:  "Global Configuration...",
			Help:  "Set global configuration options",
			Model: cfgui.NewConfigUI(cfgui.DefaultStyle(), cfgui.GlobalConfig),
		},
		{
			Name: "Local Configuration...",
			Help: "Set command specific configuration options",
		},
		{
			Separator: true,
		},
		{
			Name: "Exit",
			Help: "Exit to main menu",
		},
	}
	w := dumpui.NewModel("Wizard Debug", menu, false)

	if _, err := tea.NewProgram(w, tea.WithContext(ctx)).Run(); err != nil {
		return err
	}

	return nil
}

func debugConfigUI(ctx context.Context) error {
	return cfgui.Global(ctx)
}
