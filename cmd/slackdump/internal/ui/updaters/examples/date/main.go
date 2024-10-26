package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
)

func main() {
	var t time.Time = time.Now()
	updaters.OnClose = tea.Quit

	m := updaters.NewDTTM(&t)
	mod, err := tea.NewProgram(m).Run()
	if err != nil {
		panic(err)
	}
	_ = mod
	fmt.Printf("new value: %s\n", t)
}
