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

package apiconfig

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/bubbles/filemgr"
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
	FlagMask:   cfg.OmitAll,
	PrintFlags: true,
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
