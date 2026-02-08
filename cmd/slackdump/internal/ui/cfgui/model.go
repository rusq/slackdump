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
package cfgui

import (
	"context"
	"fmt"
	"regexp"
	"runtime/trace"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
)

const ModelID = "cfgui"

const (
	sEmpty     = "<empty>"
	sTrue      = "[x]"
	sFalse     = "[ ]"
	cursorChar = ">"
	alignGroup = ""
	alignParam = "  "

	notFound = -1
)

type Model struct {
	finished bool
	focused  bool
	cursor   int
	last     int
	state    state
	help     help.Model

	style  *Style
	keymap *Keymap

	child tea.Model
	cfgFn func() Configuration
}

func NewConfigUI(sty *Style, cfgFn func() Configuration) *Model {
	end := 0
	for _, group := range cfgFn() {
		end += len(group.Params)
	}
	end--
	return &Model{
		cfgFn:  cfgFn,
		last:   end,
		keymap: DefaultKeymap(),
		style:  sty,
		help:   help.New(),
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

type state uint8

const (
	selecting state = iota
	editing
	inline
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ctx, task := trace.NewTask(context.Background(), "cfgui.Update")
	defer task.End()

	if !m.focused {
		return m, nil
	}

	var cmds []tea.Cmd

	if _, ok := msg.(updaters.WMClose); m.child != nil && !ok && m.state != selecting {
		rgn := trace.StartRegion(ctx, "child.Update")
		child, cmd := m.child.Update(msg)
		rgn.End()
		m.child = child
		return m, cmd
	}
CASE:
	switch msg := msg.(type) {
	case updaters.WMClose:
		// child sends a close message
		if msg.WndID == updaters.ModelID {
			m.state = selecting
			m.child = nil
			cmds = append(cmds, refreshCfgCmd)
		} else if msg.WndID == ModelID {
			m.finished = true
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Up):
			if m.cursor > 0 {
				m.cursor--
			} else {
				// wrap around
				m.cursor = m.last
			}
		case key.Matches(msg, m.keymap.Down):
			if m.cursor < m.last {
				m.cursor++
			} else {
				// wrap around
				m.cursor = 0
			}
		case key.Matches(msg, m.keymap.Home):
			m.cursor = 0
		case key.Matches(msg, m.keymap.End):
			m.cursor = m.last
		case key.Matches(msg, m.keymap.Refresh):
			cmds = append(cmds, refreshCfgCmd)
		case key.Matches(msg, m.keymap.Select):
			i, j := locateParam(m.cfgFn(), m.cursor)
			if i == notFound || j == notFound {
				break CASE
			}
			if params := m.cfgFn()[i].Params[j]; params.Updater != nil {
				if params.Inline {
					m.state = inline
				} else {
					m.state = editing
				}
				m.child = params.Updater
				cmds = append(cmds, m.child.Init())
			}
		case key.Matches(msg, m.keymap.Quit):
			m.finished = true
			cmds = append(cmds, updaters.CmdClose(ModelID))
		case reNumber.MatchString(msg.String()):
			if 0 < m.cursor || m.cursor < m.last {
				m.cursor = int(msg.String()[0] - '1')
			}
		}
	}

	return m, tea.Batch(cmds...)
}

var reNumber = regexp.MustCompile(`^[1-9]$`)

func (m *Model) SetFocus(b bool) {
	m.focused = b
}

func (m *Model) IsFocused() bool {
	return m.focused
}

func (m *Model) Reset() {
	m.finished = false
	m.state = selecting
	m.child = nil
}

func (m *Model) View() string {
	_, task := trace.NewTask(context.Background(), "cfgui.View")
	defer task.End()
	if m.finished {
		return ""
	}
	sty := m.style.Focused
	if !m.focused {
		sty = m.style.Blurred
	}
	if m.child != nil && len(m.child.View()) > 0 && m.state == editing {
		return m.child.View()
	}
	return sty.Border.Render(m.view(sty))
}

func (m *Model) view(sty StyleSet) string {
	var buf strings.Builder
	line := 0
	descr := ""
	for i, group := range m.cfgFn() {
		buf.WriteString(alignGroup + sty.Title.Render(group.Name))
		buf.WriteString("\n")
		keyLen, valLen := group.maxLen()
		for j, param := range group.Params {
			selected := line == m.cursor
			if selected {
				buf.WriteString(sty.Cursor.Render(cursorChar))
				descr = m.cfgFn()[i].Params[j].Description
			} else {
				buf.WriteString(" ")
			}

			valfmt := sty.ValueDisabled
			if param.Updater != nil {
				valfmt = sty.ValueEnabled
			}

			namefmt := sty.Name
			if selected {
				namefmt = sty.SelectedName
			}
			fmt.Fprint(&buf, alignParam+namefmt.Render(fmt.Sprintf("% *s", keyLen, param.Name))+"  ")
			if selected && m.state == inline {
				buf.WriteString(m.child.View() + "\n")
			} else {
				fmt.Fprint(&buf, valfmt.Render(fmt.Sprintf("%-*s", valLen, nvl(param.Value)))+"\n")
			}
			line++
		}
	}
	if m.focused {
		buf.WriteString(alignGroup + sty.Description.Render(descr) + "\n")
		buf.WriteString(m.help.ShortHelpView(m.keymap.Bindings()))
	}

	return buf.String()
}

func nvl(s string) string {
	if s == "" {
		return sEmpty
	}
	return s
}

func (g ParamGroup) maxLen() (key int, val int) {
	for _, param := range g.Params {
		if l := len(param.Name); l > key {
			key = l
		}
		if l := len(nvl(param.Value)); l > val {
			val = l
		}
	}
	return key, val
}

func Checkbox(b bool) string {
	if b {
		return sTrue
	}
	return sFalse
}

// commands
func refreshCfgCmd() tea.Msg {
	return wmRefresh{globalConfig()}
}

type wmRefresh struct {
	cfg Configuration
}

func locateParam(cfg Configuration, line int) (int, int) {
	end := 0
	for i, group := range cfg {
		end += len(group.Params)
		if line < end {
			return i, line - (end - len(group.Params))
		}
	}
	return notFound, notFound
}
