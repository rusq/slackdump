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
package wizard

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/workspace"
)

var CmdWizard = &base.Command{
	Run:       nil, //initialised in init to prevent initialisation cycle
	UsageLine: "wiz",
	Short:     "Slackdump Wizard",
	Long: `
Slackdump Wizard guides through the dumping process.
`,
	RequireAuth: false,
}

var titlecase = cases.Title(language.English)

type menuitem struct {
	Name        string
	Description string
	cmd         *base.Command
	Submenu     *menu
}

type menu struct {
	title string
	names []string
	items []menuitem
}

func (m *menu) Add(item menuitem) {
	m.items = append(m.items, item)
	m.names = append(m.names, item.Name)
}

func init() {
	CmdWizard.Run = runWizard
}

func runWizard(ctx context.Context, cmd *base.Command, args []string) error {
	cfg.YesMan = true // to avoid interfering with the UI

	baseCommands := base.Slackdump.Commands
	if len(baseCommands) == 0 {
		panic("internal error:  no commands")
	}

	menu := makeMenu(baseCommands, "", "What would you like to do?")
	if err := show(menu, func(cmd *base.Command) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if cmd.RequireAuth {
			var err error
			ctx, err = workspace.CurrentOrNewProviderCtx(ctx)
			if err != nil {
				return err
			}
		}
		return cmd.Wizard(ctx, cmd, args)
	}); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("error running wizard: %s", err)
	}
	return nil
}

var (
	miBack = menuitem{
		Name:        "<< Back",
		Description: "",
	}
	miExit = menuitem{
		Name:        "Exit",
		Description: "",
	}
)

func makeMenu(cmds []*base.Command, parent string, title string) (m *menu) {
	m = &menu{
		title: title,
		names: make([]string, 0, len(cmds)),
		items: make([]menuitem, 0, len(cmds)),
	}
	if parent != "" {
		parent += " "
	}
	for _, cmd := range cmds {
		hasSubcommands := len(cmd.Commands) > 0
		hasWizard := cmd.Wizard != nil && !cmd.HideWizard
		isMe := strings.EqualFold(cmd.Name(), CmdWizard.Name())
		if !(hasWizard || hasSubcommands) || isMe {
			continue
		}
		name := titlecase.String(cmd.Name())
		item := menuitem{
			Name:        name,
			Description: cmd.Short,
			cmd:         cmd,
		}
		if hasSubcommands && !hasWizard {
			item.Submenu = makeMenu(cmd.Commands, name, name)
		}

		m.Add(item)
	}
	if parent != "" {
		m.Add(miBack)
	} else {
		m.Add(miExit)
	}
	return
}

func show(m *menu, onMatch func(cmd *base.Command) error) error {
	for {
		mod := newModel(m)
		p := tea.NewProgram(&mod)
		if _, err := p.Run(); err != nil {
			return err
		}
		if err := run(m, mod.val, onMatch); err != nil {
			if errors.Is(err, errBack) {
				return nil
			} else if errors.Is(err, errInvalid) {
				return err
			} else {
				return err
			}
		}
	}
}

var (
	errBack    = errors.New("back")
	errInvalid = errors.New("invalid choice")
)

func run(m *menu, choice string, onMatch func(cmd *base.Command) error) error {
	if choice == "" {
		return errBack
	}
	for _, mi := range m.items {
		if choice != mi.Name {
			continue
		}
		if mi.Submenu != nil {
			return show(mi.Submenu, onMatch)
		}
		// found
		if mi.cmd == nil { // only Exit and back won't have this.
			return errBack
		}
		return onMatch(mi.cmd)
	}
	return errInvalid
}
