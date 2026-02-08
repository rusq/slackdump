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
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/updaters"
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
