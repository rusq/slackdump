package cfgui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
)

const (
	sEmpty     = "<empty>"
	sTrue      = "[x]"
	sFalse     = "[ ]"
	cursorChar = ">"
	alignGroup = ""
	alignParam = "  "

	notFound = -1
)

type configmodel struct {
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

func NewConfigUI(sty *Style, cfgFn func() Configuration) tea.Model {
	end := 0
	for _, group := range cfgFn() {
		end += len(group.Params)
	}
	end--
	return &configmodel{
		cfgFn:  cfgFn,
		last:   end,
		keymap: DefaultKeymap(),
		style:  sty,
		help:   help.New(),
	}
}

func (m *configmodel) Init() tea.Cmd {
	return nil
}

type state uint8

const (
	selecting state = iota
	editing
	inline
)

func (m *configmodel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	var cmds []tea.Cmd

	if _, ok := msg.(updaters.WMClose); m.child != nil && !ok && m.state != selecting {
		child, cmd := m.child.Update(msg)
		m.child = child
		return m, cmd
	}

	switch msg := msg.(type) {
	case updaters.WMClose:
		// child sends a close message
		m.state = selecting
		m.child = nil
		cmds = append(cmds, refreshCfgCmd)
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
				return m, nil
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
			// child is active
			if m.state != selecting {
				break
			}
			m.finished = true
			return m, tea.Quit
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *configmodel) SetFocus(b bool) {
	m.focused = b
}

func (m *configmodel) IsFocused() bool {
	return m.focused
}

func (m *configmodel) Reset() {
	m.finished = false
	m.state = selecting
	m.child = nil
}

func (m *configmodel) View() string {
	if m.finished {
		return ""
	}
	var sty = m.style.Focused
	if !m.focused {
		sty = m.style.Blurred
	}
	if m.child != nil && len(m.child.View()) > 0 && m.state == editing {
		return m.child.View()
	}
	return sty.Border.Render(m.view(sty))
}

func (m *configmodel) view(sty StyleSet) string {
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
			fmt.Fprintf(&buf, alignParam+namefmt.Render(fmt.Sprintf("% *s", keyLen, param.Name))+"  ")
			if selected && m.state == inline {
				buf.WriteString(m.child.View() + "\n")
			} else {
				fmt.Fprintf(&buf, valfmt.Render(fmt.Sprintf("%-*s", valLen, nvl(param.Value)))+"\n")
			}
			line++
		}
	}
	buf.WriteString(alignGroup + sty.Description.Render(descr) + "\n")
	buf.WriteString(m.help.ShortHelpView(m.keymap.Bindings()))

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
