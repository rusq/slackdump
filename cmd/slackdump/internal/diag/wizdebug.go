package diag

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
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

func runWizDebug(ctx context.Context, cmd *base.Command, args []string) error {
	menu := []dumpui.MenuItem{
		{
			Name: "Run",
			Help: "Run the command",
		},
		{
			Name:  "Global Configuration...",
			Help:  "Set global configuration options",
			Model: cfgui.NewConfigUI(cfgui.DefaultStyle(), cfgui.GlobalConfig).(dumpui.FocusModel), // TODO: filthy cast
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
	w := dumpui.NewModel("Wizard Debug", menu)

	if _, err := tea.NewProgram(w).Run(); err != nil {
		return err
	}

	return nil
}
