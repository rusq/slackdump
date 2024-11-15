package menu

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
)

type Model struct {
	// Selected will be set to the selected item from the items.
	Selected  Item
	Cancelled bool

	title     string
	items     []Item
	finishing bool
	focused   bool
	preview   bool // preview child model
	Style     *Style
	Keymap    *Keymap

	help help.Model

	cursor int
	last   int
}

func New(title string, items []Item, preview bool) *Model {
	var last = len(items) - 1
	for i := last; i >= 0; i++ {
		if !items[i].Separator {
			break
		}
		last--
	}

	return &Model{
		title:     title,
		items:     items,
		Style:     DefaultStyle(),
		Keymap:    DefaultKeymap(),
		help:      help.New(),
		focused:   true,
		preview:   preview,
		finishing: false,
		cursor:    0,
		last:      last,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	child := m.items[m.cursor].Model

	if !m.focused {
		if wmclose, ok := msg.(updaters.WMClose); ok && wmclose.WndID == cfgui.ModelID {
			child.Reset()
			child.SetFocus(false)
			m.SetFocus(true)
			return m, nil
		}
		ch, cmd := child.Update(msg)
		if ch, ok := ch.(FocusModel); ok {
			m.items[m.cursor].Model = ch
		}
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.Keymap.Quit):
			m.finishing = true
			m.Cancelled = true
			m.Selected = m.items[m.cursor]
			cmds = append(cmds, tea.Quit)
		case key.Matches(msg, m.Keymap.Up):
			if m.cursor == 0 {
				m.cursor = m.last
			} else {
				for {
					if m.cursor > 0 {
						m.cursor--
					}
					if !m.items[m.cursor].Separator {
						break
					}
				}
			}
		case key.Matches(msg, m.Keymap.Down):
			if m.cursor == m.last {
				m.cursor = 0
			} else {
				for {
					if m.cursor < m.last {
						m.cursor++
					}
					if !m.items[m.cursor].Separator {
						break
					}
				}
			}
		case key.Matches(msg, m.Keymap.Select):
			validate := m.items[m.cursor].Validate
			if m.items[m.cursor].Separator || (validate != nil && validate() != nil) {
				// do nothing
			} else {
				if child := m.items[m.cursor].Model; child != nil {
					// If there is a child model, focus it.
					m.SetFocus(false)
					child.SetFocus(true)
					cmds = append(cmds, child.Init())
				} else {
					// otherwise, return selected item and quit
					m.Selected = m.items[m.cursor]
					m.finishing = true
					cmds = append(cmds, tea.Quit)
				}
			}
		}
	}
	return m, tea.Batch(cmds...)
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
	if item := m.items[m.cursor]; item.Model != nil {
		if m.focused {
			if item.Preview && m.preview {
				return lipgloss.JoinHorizontal(lipgloss.Top, m.view(), item.Model.View())
			} else {
				return m.view()
			}
		}
		return m.items[m.cursor].Model.View()
	}
	return m.view()
}

func (m *Model) Select(id string) {
	if id == m.items[m.cursor].ID {
		return
	}
	for i, item := range m.items {
		if item.ID == id && !item.Separator {
			m.cursor = i
			break
		}
	}
}

func capfirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func (m *Model) view() string {
	var b strings.Builder

	sty := m.Style.Focused
	if !m.focused {
		sty = m.Style.Blurred
	}

	currentItem := m.items[m.cursor]
	currentDisabled := currentItem.Validate != nil && currentItem.Validate() != nil

	p := b.WriteString
	// Header
	p(sty.Title.Render(m.title) + "\n")
	if currentDisabled {
		p(sty.Description.Render("Requirements not met: " + capfirst(currentItem.Validate().Error())))
	} else {
		p(sty.Description.Render(m.items[m.cursor].Help))
	}
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

		var (
			current  = i == m.cursor
			disabled = itm.Validate != nil && itm.Validate() != nil
		)
		if disabled {
			p(sty.ItemDisabled.Render(iftrue(current, pointer, padding) + itm.Name))
			continue
		}
		if current {
			p(sty.Cursor.Render(pointer) + sty.ItemSelected.Render(itm.Name))
		} else {
			p(sty.Item.Render(padding + itm.Name))
		}
	}
	p("\n" + m.footer())
	return sty.Border.Render(b.String())
}

func iftrue(t bool, a, b string) string {
	if t {
		return a
	}
	return b
}

func (m *Model) footer() string {
	return m.help.ShortHelpView(m.Keymap.Bindings())
}
