package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v3/internal/source"
)

func main() {
	var result source.StorageType
	updaters.OnClose = tea.Quit

	l := updaters.NewPicklist(&result, huh.NewSelect[source.StorageType]().
		Title("Title").
		Options(
			huh.NewOption("None", source.STnone),
			huh.NewOption("Standard", source.STstandard),
			huh.NewOption("Mattermost", source.STmattermost),
		))

	if _, err := tea.NewProgram(l).Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("selected: ", result)
}
