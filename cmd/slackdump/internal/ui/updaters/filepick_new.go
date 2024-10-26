package updaters

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

type FileNewModel struct {
	entry      StringModel
	cnfrm      *huh.Form
	allowOvwr  bool
	confirming bool
	finishing  bool
}

func NewFileNew(v *string, placeholder string, showPrompt bool, overwrite bool) FileNewModel {
	m := FileNewModel{
		entry:     NewString(v, placeholder, showPrompt, ui.ValidateNotExists),
		cnfrm:     newConfirmForm(),
		allowOvwr: overwrite,
	}
	return m
}

func newConfirmForm() *huh.Form {
	f := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("File already exists").Description("Do you want to overwrite it?").Key("confirm")))
	f.CancelCmd = func() tea.Msg { return fwmCancel }
	f.SubmitCmd = func() tea.Msg { return fwmConfirm }
	return f
}

// form callback message type
type formMsg int

// form callback messages
const (
	fwmConfirm formMsg = iota
	fwmCancel
)

func (m FileNewModel) Init() tea.Cmd {
	return tea.Batch(m.entry.Init(), m.cnfrm.Init())
}

func (m FileNewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, OnClose
		case "esc":
			if m.confirming {
				m.confirming = false
				return m, tea.Batch(m.entry.Init(), m.cnfrm.Init())
			}
			m.finishing = true
			return m, OnClose
		case "enter":
			if !m.allowOvwr {
				return m, nil
			}
			if m.entry.Err() != nil && !m.confirming {
				m.cnfrm = newConfirmForm()
				m.confirming = true
				return m, tea.Batch(m.cnfrm.Init())
			}
		}

	case formMsg:
		// form was submitted.
		m.confirming = false
		switch msg {
		case fwmCancel:
			return m, tea.Batch(m.entry.Init(), m.cnfrm.Init())
		case fwmConfirm:
			if m.cnfrm.GetBool("confirm") {
				*m.entry.Value = m.entry.m.Value()
				m.finishing = true
				return m, OnClose
			} else {
				return m, tea.Batch(m.entry.Init(), m.cnfrm.Init())
			}
		}

	}

	if m.confirming {
		mod, cmd := m.cnfrm.Update(msg)
		m.cnfrm = mod.(*huh.Form)
		cmds = append(cmds, cmd)
	} else {
		mod, cmd := m.entry.Update(msg)
		m.entry = mod.(StringModel)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m FileNewModel) View() string {
	if m.finishing {
		return ""
	}
	var buf strings.Builder
	buf.WriteString(m.entry.View())
	if m.confirming {
		buf.WriteString("\n\n" + m.cnfrm.View())
	}
	return buf.String()
}
