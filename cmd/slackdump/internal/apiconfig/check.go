package apiconfig

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/filemgr"
)

var CmdConfigCheck = &base.Command{
	UsageLine: "slackdump config check",
	Short:     "validate the existing config for errors",
	Long: `
# Config Check Command

Allows to check the config for errors and invalid values.

Example:

    slackdump config check myconfig.yaml

It will check for duplicate and unknown keys, and also ensure that values are
within the allowed boundaries.
`,
}

func init() {
	CmdConfigCheck.Run = runConfigCheck
	CmdConfigCheck.Wizard = wizConfigCheck
}

func runConfigCheck(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) == 0 || args[0] == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("config filename must be specified")
	}
	filename := args[0]
	if err := checkFile(filename); err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	fmt.Printf("Config file %q: OK\n", filename)
	return nil
}

func checkFile(filename string) error {
	if _, err := Load(filename); err != nil {
		return fmt.Errorf("config file %q not OK: %s", filename, err)
	}
	return nil
}

func wizConfigCheck(ctx context.Context, cmd *base.Command, args []string) error {
	f := filemgr.NewModel(".", 15, "*.yaml", "*.yml")
	f.ShowHelp = true
	f.Debug = os.Getenv("DEBUG") != ""
	f.Style = filemgr.Style{
		Normal:    cfg.Theme.Focused.File,
		Directory: cfg.Theme.Focused.Directory,
		Inverted: lipgloss.NewStyle().
			Foreground(cfg.Theme.Focused.FocusedButton.GetForeground()).
			Background(cfg.Theme.Focused.FocusedButton.GetBackground()),
	}
	vp := viewport.New(80-filemgr.Width, f.Height)
	vp.Style = lipgloss.NewStyle().Border(lipgloss.DoubleBorder(), true).Margin(0, 2)
	m := checkerModel{
		files: f,
		view:  vp,
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		return err
	}

	return nil
}
