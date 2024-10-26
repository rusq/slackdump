package cfgui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
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

type Style struct {
	Border        lipgloss.Style
	Title         lipgloss.Style
	Description   lipgloss.Style
	Name          lipgloss.Style
	ValueEnabled  lipgloss.Style
	ValueDisabled lipgloss.Style
	SelectedName  lipgloss.Style
	Cursor        lipgloss.Style
}

type configmodel struct {
	finished bool
	cfgFn    func() Configuration
	cursor   int
	end      int
	Style    Style

	child tea.Model
	state state
}

func (m configmodel) Init() tea.Cmd {
	return nil
}

type state uint8

const (
	selecting state = iota
	editing
	inline
)

func (m configmodel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				// wrap around
				m.cursor = m.end
			}
		case "down", "j":
			if m.cursor < m.end {
				m.cursor++
			} else {
				// wrap around
				m.cursor = 0
			}
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = m.end
		case "f5":
			cmds = append(cmds, refreshCfgCmd)
		case "enter":
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
		case "q", "esc", "ctrl+c":
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

func (m configmodel) View() string {
	if m.finished {
		return ""
	}
	if m.child != nil && len(m.child.View()) > 0 && m.state == editing {
		return m.child.View()
	}
	return ui.DefaultTheme().Focused.Border.Render(m.view())
}

func (m configmodel) view() string {
	var buf strings.Builder
	line := 0
	descr := ""
	for i, group := range m.cfgFn() {
		buf.WriteString(alignGroup + m.Style.Title.Render(group.Name))
		buf.WriteString("\n")
		keyLen, valLen := group.maxLen()
		for j, param := range group.Params {
			selected := line == m.cursor
			if selected {
				buf.WriteString(m.Style.Cursor.Render(cursorChar))
				descr = m.cfgFn()[i].Params[j].Description
			} else {
				buf.WriteString(" ")
			}

			valfmt := m.Style.ValueDisabled
			if param.Updater != nil {
				valfmt = m.Style.ValueEnabled
			}

			namefmt := m.Style.Name
			if selected {
				namefmt = m.Style.SelectedName
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
	buf.WriteString(alignGroup + m.Style.Description.Render(descr))

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
