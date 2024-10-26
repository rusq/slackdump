package dumpui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

type Model struct {
	Title     string
	Items     []MenuItem
	finishing bool
	focused   bool
	Style     *Style
	Keymap    *Keymap

	help help.Model

	cursor int
}

type Style struct {
	Focused StyleSet
	Blurred StyleSet
}

type StyleSet struct {
	Border       lipgloss.Style
	Title        lipgloss.Style
	Description  lipgloss.Style
	Cursor       lipgloss.Style
	Item         lipgloss.Style
	ItemSelected lipgloss.Style
	ItemDisabled lipgloss.Style
}

func DefaultStyle() *Style {
	t := ui.DefaultTheme()
	return &Style{
		Focused: StyleSet{
			Border:       t.Focused.Border,
			Title:        t.Focused.Title,
			Description:  t.Focused.Description,
			Cursor:       t.Focused.Cursor,
			Item:         t.Focused.Text,
			ItemSelected: t.Focused.Selected,
			ItemDisabled: t.Blurred.Text,
		},
		Blurred: StyleSet{
			Border:       t.Blurred.Border,
			Title:        t.Blurred.Title,
			Description:  t.Blurred.Description,
			Cursor:       t.Blurred.Cursor,
			Item:         t.Blurred.Text,
			ItemSelected: t.Blurred.Selected,
			ItemDisabled: t.Blurred.Text,
		},
	}
}

type Keymap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Quit   key.Binding
}

func DefaultKeymap() *Keymap {
	return &Keymap{
		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑", "up")),
		Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓", "down")),
		Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func NewModel(title string, items []MenuItem) *Model {
	return &Model{
		Title:     title,
		Items:     items,
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
				if !m.Items[m.cursor].Disabled && !m.Items[m.cursor].Separator {
					break
				}
			}
		case key.Matches(msg, m.Keymap.Down):
			for {
				if m.cursor < len(m.Items)-1 {
					m.cursor++
				}
				if !m.Items[m.cursor].Disabled && !m.Items[m.cursor].Separator {
					break
				}
			}
		case key.Matches(msg, m.Keymap.Select):
			if m.Items[m.cursor].Disabled || m.Items[m.cursor].Separator {
				return m, nil
			}
			m.finishing = true
			// TODO: return the selected choice
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) SetFocus(b bool) {
	m.focused = b
}

func (m *Model) View() string {
	if m.finishing {
		return ""
	}
	var b strings.Builder

	sty := m.Style.Focused
	if !m.focused {
		sty = m.Style.Blurred
	}
	p := b.WriteString
	// Header
	p(sty.Title.Render(m.Title) + "\n")
	p(sty.Description.Render(m.Items[m.cursor].Help))
	const (
		padding = "  "
		pointer = "> "
	)
	for i, itm := range m.Items {
		p("\n")
		if itm.Separator {
			p(padding + ui.MenuSeparator)
			continue
		}
		if itm.Disabled {
			p(sty.ItemDisabled.Render(padding + itm.Name))
			continue
		}
		if i == m.cursor {
			p(sty.Cursor.Render(pointer) + sty.ItemSelected.Render(itm.Name))
		} else {
			if itm.Disabled {
				p(sty.ItemDisabled.Render(padding + itm.Name))
			} else {
				p(sty.Item.Render(padding + itm.Name))
			}
		}
	}
	b.WriteString("\n\n" + m.footer())
	return sty.Border.Render(b.String())
}

func (m *Model) footer() string {
	return m.help.ShortHelpView([]key.Binding{m.Keymap.Up, m.Keymap.Down, m.Keymap.Select, m.Keymap.Quit})
}

type MenuItem struct {
	Name      string
	Help      string
	Disabled  bool
	Separator bool
}
