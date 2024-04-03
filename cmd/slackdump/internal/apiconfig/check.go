package apiconfig

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
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
	if _, err := Load(filename); err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("config file %q not OK: %s", filename, err)
	}
	fmt.Printf("Config file %q: OK\n", filename)
	return nil
}

func wizConfigCheck(ctx context.Context, cmd *base.Command, args []string) error {
	fp := ui.NewFilePicker("Select a config file to check", ".", "yaml", "yml")
	ret, err := tea.NewProgram(fp).Run()
	if err != nil {
		return err
	}
	fp = ret.(ui.FileSystemModel)

	return runConfigCheck(ctx, cmd, []string{fp.SelectedFile})
}
