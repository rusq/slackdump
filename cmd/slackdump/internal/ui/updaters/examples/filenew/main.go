package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
)

func main() {
	var s string = "main.go"
	updaters.OnClose = tea.Quit
	m := updaters.NewFileNew(&s, "enter filename", true, true)
	mod, err := tea.NewProgram(m).Run()
	if err != nil {
		panic(err)
	}
	_ = mod
	fmt.Printf("new value: %s\n", s)
}
