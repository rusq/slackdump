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

package cfgui

import (
	"github.com/charmbracelet/bubbles/key"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
)

type Keymap struct {
	Up      key.Binding
	Down    key.Binding
	Home    key.Binding
	End     key.Binding
	Refresh key.Binding
	Select  key.Binding
	Quit    key.Binding
}

func DefaultKeymap() *Keymap {
	return &Keymap{
		Up:      ui.KeyUpBinding(),
		Down:    ui.KeyDownBinding(),
		Home:    ui.KeyHomeBinding(),
		End:     ui.KeyEndBinding(),
		Refresh: ui.KeyRefreshBinding(),
		Select:  ui.KeySelectBinding("submit"),
		Quit:    ui.KeyQuitBinding(),
	}
}

func (k *Keymap) Bindings() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Home, k.Refresh, k.Select, k.Quit}
}
