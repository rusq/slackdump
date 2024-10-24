package updaters

import tea "github.com/charmbracelet/bubbletea"

type WMClose = struct{}

// OnClose defines the global command to close the program.  It is set
// by default to [CmdClose], but if running standalone, one must set it
// to [tea.Quit], otherwise the program will not exit.
var OnClose = CmdClose

func CmdClose() tea.Msg {
	return WMClose{}
}
