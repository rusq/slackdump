package cfgui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
)

const (
	sEmpty     = "<empty>"
	sTrue      = "[x]"
	sFalse     = "[ ]"
	cursor     = ">"
	alignGroup = ""
	alignParam = "  "
)

func New() configmodel {
	cfg := effectiveConfig()
	end := 0
	for _, group := range cfg {
		end += len(group.params)
	}
	end--
	return configmodel{
		cfg: effectiveConfig(),
		end: end,
	}
}

func Show(ctx context.Context) error {
	p := tea.NewProgram(New())
	_, err := p.Run()
	return err
}

type configmodel struct {
	finished bool
	cfg      configuration
	cursor   int
	end      int

	child tea.Model
}

func (m configmodel) Init() tea.Cmd {
	return nil
}

func (m configmodel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case closeMsg:
		// child sends a close message
		m.child = nil
		cmds = append(cmds, refreshCfgCmd)
	case refreshMsg:
		m.cfg = msg.cfg
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				// wrap around
				m.cursor = m.end - 1
			}
		case "down", "j":
			if m.cursor < m.end-1 {
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
			i, j := locateParam(m.cfg, m.cursor)
			if i == -1 || j == -1 {
				return m, nil
			}
			if m.cfg[i].params[j].Model != nil {
				m.child = m.cfg[i].params[j].Model
			}
		case "q", "esc", "ctrl+c":
			m.finished = true
			return m, tea.Quit
		}
	}

	if m.child != nil {
		_, cmd := m.child.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m configmodel) View() string {
	if m.finished {
		return ""
	}
	if m.child != nil {
		return m.child.View()
	}

	// formatting functions
	var (
		sGrp   = cfg.Theme.Focused.Title.Render
		sKey   = cfg.Theme.Focused.SelectedOption.Render
		sVal   = cfg.Theme.Focused.UnselectedOption.Render
		sDescr = cfg.Theme.Focused.Description.Render
	)

	var buf strings.Builder
	line := 0
	descr := ""
	for i, group := range m.cfg {
		buf.WriteString(alignGroup + sGrp(group.name))
		buf.WriteString("\n")
		keyLen, valLen := group.maxLen()
		for j, param := range group.params {
			if line == m.cursor {
				buf.WriteString(cursor)
				descr = m.cfg[i].params[j].Description
			} else {
				buf.WriteString(" ")
			}
			fmt.Fprintf(&buf, alignParam+
				sKey(fmt.Sprintf("% *s", keyLen, param.Name))+"  "+
				sVal(fmt.Sprintf("%-*s", valLen, nvl(param.Value)))+"\n",
			)
			line++
		}
	}
	buf.WriteString(alignGroup + sDescr(descr))

	return buf.String()
}

func nvl(s string) string {
	if s == "" {
		return sEmpty
	}
	return s
}

func (g group) maxLen() (key int, val int) {
	for _, param := range g.params {
		if l := len(param.Name); l > key {
			key = l
		}
		if l := len(nvl(param.Value)); l > val {
			val = l
		}
	}
	return key, val
}

func checkbox(b bool) string {
	if b {
		return sTrue
	}
	return sFalse
}

// commands
func refreshCfgCmd() tea.Msg {
	return refreshMsg{effectiveConfig()}
}

type refreshMsg struct {
	cfg configuration
}

type closeMsg = struct{}

func cmdClose() tea.Msg {
	return closeMsg{}
}

func locateParam(cfg configuration, line int) (int, int) {
	end := 0
	for i, group := range cfg {
		end += len(group.params)
		if line < end {
			return i, line - (end - len(group.params))
		}
	}
	return -1, -1
}
