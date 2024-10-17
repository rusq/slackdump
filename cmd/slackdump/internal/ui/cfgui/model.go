package cfgui

import (
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

	notFound = -1
)

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

	if _, ok := msg.(wmClose); m.child != nil && !ok {
		child, cmd := m.child.Update(msg)
		m.child = child
		return m, cmd
	}

	switch msg := msg.(type) {
	case wmClose:
		// child sends a close message
		m.child = nil
		cmds = append(cmds, refreshCfgCmd)
	case wmRefresh:
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
			if i == notFound || j == notFound {
				return m, nil
			}
			if m.cfg[i].params[j].Model != nil {
				m.child = m.cfg[i].params[j].Model
				cmds = append(cmds, m.child.Init())
			}
		case "q", "esc", "ctrl+c":
			// child is active
			if m.child != nil {
				break
			}
			m.finished = true
			return m, tea.Quit
		}
	}

	return m, tea.Batch(cmds...)
}

// formatting functions
var (
	fmtgrp       = cfg.Theme.Focused.Title.Render
	fmtname      = cfg.Theme.Focused.SelectedOption.Render
	fmtvalactive = cfg.Theme.Focused.UnselectedOption.Render
	fmtvalinact  = cfg.Theme.Focused.Description.Render
	fmtdescr     = cfg.Theme.Focused.Description.Render
)

func (m configmodel) View() string {
	if m.finished {
		return ""
	}
	if m.child != nil {
		return m.child.View()
	}

	var buf strings.Builder
	line := 0
	descr := ""
	for i, group := range m.cfg {
		buf.WriteString(alignGroup + fmtgrp(group.name))
		buf.WriteString("\n")
		keyLen, valLen := group.maxLen()
		for j, param := range group.params {
			if line == m.cursor {
				buf.WriteString(cursor)
				descr = m.cfg[i].params[j].Description
			} else {
				buf.WriteString(" ")
			}
			valfmt := fmtvalinact
			if param.Model != nil {
				valfmt = fmtvalactive
			}

			fmt.Fprintf(&buf, alignParam+
				fmtname(fmt.Sprintf("% *s", keyLen, param.Name))+"  "+
				valfmt(fmt.Sprintf("%-*s", valLen, nvl(param.Value)))+"\n",
			)
			line++
		}
	}
	buf.WriteString(alignGroup + fmtdescr(descr))

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
	return wmRefresh{effectiveConfig()}
}

type wmRefresh struct {
	cfg configuration
}

type wmClose = struct{}

func cmdClose() tea.Msg {
	return wmClose{}
}

func locateParam(cfg configuration, line int) (int, int) {
	end := 0
	for i, group := range cfg {
		end += len(group.params)
		if line < end {
			return i, line - (end - len(group.params))
		}
	}
	return notFound, notFound
}
