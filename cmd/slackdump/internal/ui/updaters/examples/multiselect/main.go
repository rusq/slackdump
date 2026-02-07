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
