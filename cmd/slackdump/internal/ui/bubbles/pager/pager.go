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

// Package pager provides a scrollable read-only view for pre-rendered text,
// similar to a system pager.
package pager

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
)

type Keymap struct {
	Up     key.Binding
	Down   key.Binding
	Page   key.Binding
	HomeNd key.Binding
	Quit   key.Binding
}

// DefaultKeymap returns the pager key bindings.  Up, Down and Page are for
// help display only: scrolling for those keys is handled by the embedded
// viewport's own default keymap, so they must stay in sync with it (guarded
// by TestDefaultKeymap_advertisedKeysHandledByViewport).
func DefaultKeymap() *Keymap {
	return &Keymap{
		Up:     ui.KeyUpBinding(),
		Down:   ui.KeyDownBinding(),
		Page:   key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("PgUp/PgDn", "page")),
		HomeNd: key.NewBinding(key.WithKeys("home", "end"), key.WithHelp("Home/End", "top/bottom")),
		Quit:   ui.KeyQuitBinding(),
	}
}

func (k *Keymap) Bindings() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Page, k.HomeNd, k.Quit}
}

// Model is a scrollable pager for a pre-rendered string.
type Model struct {
	title   string
	content string

	vp        viewport.Model
	help      help.Model
	keymap    *Keymap
	ready     bool
	finishing bool
}

// New creates a new pager with the given title and pre-rendered content.
// The viewport is sized when the first tea.WindowSizeMsg arrives.
func New(title string, content string) *Model {
	return &Model{
		title:   title,
		content: content,
		help:    ui.NewHelp(),
		keymap:  DefaultKeymap(),
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

// chrome is the number of lines occupied by the header and the footer.
const chrome = 2

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			m.vp = viewport.New(msg.Width, msg.Height-chrome)
			m.vp.SetContent(m.content)
			m.ready = true
		} else {
			m.vp.Width = msg.Width
			m.vp.Height = msg.Height - chrome
			m.vp.SetYOffset(m.vp.YOffset) // clamp to the new size
		}
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Quit):
			m.finishing = true
			return m, tea.Quit
		case key.Matches(msg, m.keymap.HomeNd):
			if msg.String() == "home" {
				m.vp.GotoTop()
			} else {
				m.vp.GotoBottom()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	if m.finishing {
		return ""
	}
	sty := ui.DefaultTheme().Focused
	if !m.ready {
		return sty.Description.Render("loading...")
	}
	return sty.Title.Render(m.title) + "\n" + m.vp.View() + "\n" + m.footer()
}

func (m *Model) footer() string {
	percent := fmt.Sprintf("%3.0f%%", m.vp.ScrollPercent()*100)
	return ui.DefaultTheme().Focused.Description.Render(percent) + " " + m.help.ShortHelpView(m.keymap.Bindings())
}
