package apiconfig

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/filemgr"
)

var CmdConfigCheck = &base.Command{
	UsageLine: "slackdump config check",
	Short:     "validate the existing config for errors",
	Long: `
# Check Command

Validates the configuration file for errors and invalid values.

Example:

    slackdump config check myconfig.toml

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
	if err := CheckFile(filename); err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	fmt.Printf("Config file %q: OK\n", filename)
	return nil
}

func CheckFile(filename string) error {
	if _, err := Load(filename); err != nil {
		return fmt.Errorf("config file %q not OK: %s", filename, err)
	}
	return nil
}

func wizConfigCheck(ctx context.Context, cmd *base.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	f := filemgr.New(os.DirFS(cwd), cwd, ".", 15, ConfigExts...)
	f.Focus()
	f.ShowHelp = true
	f.Style = filemgr.Style{
		Normal:    ui.DefaultTheme().Focused.UnselectedFile,
		Directory: ui.DefaultTheme().Focused.Directory,
		Inverted:  ui.DefaultTheme().Focused.SelectedFile,
		Shaded:    ui.DefaultTheme().Focused.DisabledFile,
		CurDir:    ui.DefaultTheme().Focused.Description,
	}
	vp := viewport.New(80-filemgr.Width, f.Height)
	vp.Style = lipgloss.NewStyle().Margin(0, 2)
	vp.SetContent("Select a config file to check and press [Enter].")
	m := checkerModel{
		files:      f,
		view:       vp,
		FocusStyle: ui.DefaultTheme().Focused.Border,
		BlurStyle:  ui.DefaultTheme().Blurred.Border,
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		return err
	}

	return nil
}
