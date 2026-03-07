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

// Package dumpui provides a universal wizard for running dump-family commands.
package dumpui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/bubbles/menu"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/cfgui"
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
	// Help is the markdown help text.
	Help string
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
	var menu = func() *menu.Model {
		var items []menu.Item
		if w.LocalConfig != nil {
			items = append(items, menu.Item{
				ID:      actLocalConfig,
				Name:    w.Name + " Options...",
				Help:    description[actLocalConfig],
				Preview: true,
				Model:   cfgui.NewConfigUI(cfgui.DefaultStyle(), w.LocalConfig),
			})
		}

		items = append(
			items,
			menu.Item{
				ID:   actRun,
				Name: "Run " + w.Name,
				Help: description[actRun],
				Validate: func() error {
					if w.ValidateParamsFn != nil {
						return w.ValidateParamsFn()
					}
					return nil
				},
			},
		)
		if w.Help != "" {
			items = append(items, menu.Item{
				ID:   "help",
				Name: "Help",
				Help: "Read help for " + w.Name,
			})
		}

		items = append(items,
			menu.Item{Separator: true},
			menu.Item{
				ID:    actGlobalConfig,
				Name:  "Global Configuration...",
				Help:  description[actGlobalConfig],
				Model: cfgui.NewConfigUI(cfgui.DefaultStyle(), cfgui.GlobalConfig), // TODO: filthy cast
			},
			menu.Item{Separator: true},
			menu.Item{ID: actExit, Name: "Exit", Help: description[actExit]},
		)

		return menu.New(w.Title, items, true)
	}

LOOP:
	for {
		m := menu()
		if _, err := tea.NewProgram(m, tea.WithContext(ctx)).Run(); err != nil {
			return err
		}
		if m.Cancelled {
			break
		}
		switch m.Selected.ID {
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
		case actExit:
			break LOOP
		}
	}

	return nil
}
