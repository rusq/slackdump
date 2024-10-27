package dumpui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

type Model struct {
	// Selected will be set to the selected item from the items.
	Selected MenuItem

	title     string
	items     []MenuItem
	finishing bool
	focused   bool
	Style     *Style
	Keymap    *Keymap

	help help.Model

	cursor int
}

func NewModel(title string, items []MenuItem) *Model {
	return &Model{
		title:     title,
		items:     items,
		Style:     DefaultStyle(),
		Keymap:    DefaultKeymap(),
		help:      help.New(),
		focused:   true,
		finishing: false,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	child := m.items[m.cursor].Model
	cfg.Log.Debugf("msg: %v, child is nil? %t", msg, child == nil)

	if child != nil && child.IsFocused() {
		ch, cmd := child.Update(msg)
		m.items[m.cursor].Model = ch.(FocusModel)
		if cmd != nil && cmd() != nil {
			if _, ok := cmd().(tea.QuitMsg); ok {
				// if child quit, we need to set focus back to the menu.
				m.SetFocus(true)
				child.SetFocus(false)
				child.Reset()
				return m, nil
			}
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.Keymap.Quit):
			m.finishing = true
			return m, tea.Quit
		case key.Matches(msg, m.Keymap.Up):
			for {
				if m.cursor > 0 {
					m.cursor--
				}
				if !m.items[m.cursor].Separator {
					break
				}
			}
		case key.Matches(msg, m.Keymap.Down):
			for {
				if m.cursor < len(m.items)-1 {
					m.cursor++
				}
				if !m.items[m.cursor].Separator {
					break
				}
			}
		case key.Matches(msg, m.Keymap.Select):
			dfn := m.items[m.cursor].IsDisabled
			if m.items[m.cursor].Separator || (dfn != nil && dfn()) {
				return m, nil
			}
			m.Selected = m.items[m.cursor]

			if child := m.items[m.cursor].Model; child != nil {
				m.SetFocus(false)
				child.SetFocus(true)
				return m, nil
			}

			m.finishing = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) SetFocus(b bool) {
	m.focused = b
}

func (m *Model) IsFocused() bool {
	return m.focused
}

func (m *Model) View() string {
	if m.finishing {
		return ""
	}
	if m.items[m.cursor].Model != nil {
		return lipgloss.JoinHorizontal(lipgloss.Top, m.view(), m.items[m.cursor].Model.View())
	}
	return m.view()
}

func (m *Model) view() string {
	var b strings.Builder

	sty := m.Style.Focused
	if !m.focused {
		sty = m.Style.Blurred
	}
	p := b.WriteString
	// Header
	p(sty.Title.Render(m.title) + "\n")
	p(sty.Description.Render(m.items[m.cursor].Help))
	const (
		padding = "  "
		pointer = "> "
	)
	for i, itm := range m.items {
		p("\n")
		if itm.Separator {
			p(padding + ui.MenuSeparator)
			continue
		}

		if itm.IsDisabled != nil && itm.IsDisabled() {
			p(sty.ItemDisabled.Render(padding + itm.Name))
			continue
		}
		if i == m.cursor {
			p(sty.Cursor.Render(pointer) + sty.ItemSelected.Render(itm.Name))
		} else {
			p(sty.Item.Render(padding + itm.Name))
		}
	}
	b.WriteString("\n" + m.footer())
	return sty.Border.Render(b.String())
}

func (m *Model) footer() string {
	return m.help.ShortHelpView(m.Keymap.Bindings())
}
