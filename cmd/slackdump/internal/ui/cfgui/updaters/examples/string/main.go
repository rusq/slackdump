package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui/updaters"
)

func main() {
	var s string = "previous value"
	updaters.OnClose = tea.Quit
	m := updaters.NewString(&s, ui.ValidateNotExists)
	mod, err := tea.NewProgram(m).Run()
	if err != nil {
		panic(err)
	}
	_ = mod
	fmt.Printf("new value: %s\n", s)
}
