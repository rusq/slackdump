package apiconfig

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	if _, err := Load(filename); err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("config file %q not OK: %s", filename, err)
	}
	fmt.Printf("Config file %q: OK\n", filename)
	return nil
}

func wizConfigCheck(ctx context.Context, cmd *base.Command, args []string) error {
	f := filemgr.NewModel("*.yaml", ".")
	f.Height = 8
	m := checkerModel{
		files: f,
		view:  viewport.New(40, f.Height+2),
	}
	_, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}

	// return runConfigCheck(ctx, cmd, []string{fp.files})
	return nil
}

type checkerModel struct {
	files     filemgr.Model
	view      viewport.Model
	finishing bool
}

func (m checkerModel) Init() tea.Cmd {
	return tea.Batch(m.files.Init(), m.view.Init())
}

func (m checkerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.finishing = true
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.files, cmd = m.files.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.view, cmd = m.view.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m checkerModel) View() string {
	if m.finishing {
		return ""
	}
	var buf strings.Builder
	fmt.Fprintf(&buf, "%s\n%s", m.view.View(), m.files.View())
	return buf.String()
}
