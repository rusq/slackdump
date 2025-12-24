package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
)

func main() {
	var result []string
	updaters.OnClose = tea.Quit

	l := updaters.NewMultiSelect(&result, huh.NewMultiSelect[string]().
		Title("Title").
		Options(
			huh.NewOption("DMs", "im").Selected(true),
			huh.NewOption("Group Messages", "mpim"),
			huh.NewOption("Public Channels", "public_channel"),
			huh.NewOption("Private Channel", "private_channel"),
		))

	if _, err := tea.NewProgram(l).Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("selected: ", result)
}
