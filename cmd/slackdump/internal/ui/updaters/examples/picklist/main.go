package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
)

func main() {
	var result fileproc.StorageType

	updaters.OnClose = tea.Quit

	l := updaters.NewPicklist(&result, huh.NewSelect[fileproc.StorageType]().
		Title("Title").
		Description("Description").
		Options(
			huh.NewOption("None", fileproc.STnone),
			huh.NewOption("Standard", fileproc.STstandard),
			huh.NewOption("Mattermost", fileproc.STmattermost),
		))

	if _, err := tea.NewProgram(l).Run(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("selected: ", result)
}
