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

package updaters

import tea "github.com/charmbracelet/bubbletea"

const ModelID = "updater"

type WMClose struct {
	// WndID is the window ID to close.  If empty, the current window
	// will be closed.
	WndID string
}

// OnClose defines the global command to close the program.  It is set
// by default to [CmdClose], but if running standalone, one must set it
// to [tea.Quit], otherwise the program will not exit.
var OnClose = CmdClose(ModelID)

func CmdClose(id string) func() tea.Msg {
	return func() tea.Msg {
		return WMClose{id}
	}
}

// WMError is sent when an error occurs, for example, a validation error,
// so that caller can display the error message.
type WMError error

// CmdError sends an error message.
func CmdError(err error) tea.Msg {
	return err
}
