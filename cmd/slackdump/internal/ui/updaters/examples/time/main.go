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
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/bubbles/btime"
)

const logname = "time.log"

func main() {
	logf, err := os.Create(logname)
	if err != nil {
		log.Fatal(err)
	}
	defer logf.Close()
	log.SetOutput(logf)

	m := btime.New(time.Now())
	m.Focused = true
	m.ShowHelp = true
	p, err := tea.NewProgram(timeModel{m}).Run()
	if err != nil {
		log.Fatal(err)
	}
	_ = p
}

type timeModel struct {
	*btime.Model
}

// Update wraps the Update method of the embedded btime.Model, to satisfy
// the tea.Model interface.
func (m timeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	mod, cmd := m.Model.Update(msg)
	m.Model = mod
	return m, cmd
}
