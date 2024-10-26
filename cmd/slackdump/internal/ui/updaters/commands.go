package updaters

import tea "github.com/charmbracelet/bubbletea"

type WMClose struct{}

// OnClose defines the global command to close the program.  It is set
// by default to [CmdClose], but if running standalone, one must set it
// to [tea.Quit], otherwise the program will not exit.
var OnClose = CmdClose

func CmdClose() tea.Msg {
	return WMClose{}
}

// WMError is sent when an error occurs, for example, a validation error,
// so that caller can display the error message.
type WMError error

// CmdError sends an error message.
func CmdError(err error) tea.Msg {
	return err
}
