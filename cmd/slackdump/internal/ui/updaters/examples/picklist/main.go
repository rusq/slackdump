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
package main

import (
	"fmt"
	"log"

	"github.com/rusq/slackdump/v4/source"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/updaters"
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
