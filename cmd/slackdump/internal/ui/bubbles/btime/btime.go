package btime

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// KeyMap is the key bindings for different actions within the datepicker.
type KeyMap struct {
	Up        key.Binding
	Right     key.Binding
	Down      key.Binding
	Left      key.Binding
	Backspace key.Binding
	Delete    key.Binding
	FocusPrev key.Binding
	FocusNext key.Binding
	Quit      key.Binding
}

type Model struct {
	Time   time.Time
	entry  [6]int
	maxnum [3]int
	cursor int

	KeyMap   KeyMap
	Styles   Styles
	ShowHelp bool

	Focused   bool
	finishing bool
}

type Styles struct {
	Selected   lipgloss.Style
	Unselected lipgloss.Style
	Separator  lipgloss.Style
	Help       lipgloss.Style
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k", "+")),
		Right:     key.NewBinding(key.WithKeys("right", "l")),
		Down:      key.NewBinding(key.WithKeys("down", "j", "-")),
		Left:      key.NewBinding(key.WithKeys("left", "h")),
		Backspace: key.NewBinding(key.WithKeys("backspace", "ctrl+h")),
		Delete:    key.NewBinding(key.WithKeys("delete", "x")),
		FocusPrev: key.NewBinding(key.WithKeys("shift+tab")),
		FocusNext: key.NewBinding(key.WithKeys("tab")),
		Quit:      key.NewBinding(key.WithKeys("ctrl+c", "q")),
	}
}

func DefaultStyles() Styles {
	return Styles{
		Selected:   lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		Unselected: lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
		Separator:  lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Help:       lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	}
}

func New(t time.Time) *Model {
	tm := &Model{
		Time:     t,
		entry:    [6]int{0, 0, 0, 0, 0, 0},
		maxnum:   [3]int{23, 59, 59},
		cursor:   0,
		Focused:  false,
		KeyMap:   DefaultKeyMap(),
		Styles:   DefaultStyles(),
		ShowHelp: false,
	}
	tm.toEntry()
	return tm
}

func (m *Model) Focus() {
	m.Focused = true
}

func (m *Model) Blur() {
	m.Focused = false
}

func (m *Model) SetTime(t time.Time) {
	m.Time = t
	m.toEntry()
}

func (m *Model) Init() tea.Cmd {
	return nil
}

var digitsRe = regexp.MustCompile(`\d`)

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	if !m.Focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Quit):
			m.finishing = true
			m.updateTime()
			return m, tea.Quit
		case key.Matches(msg, m.KeyMap.Left):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.KeyMap.Right):
			if m.cursor < len(m.entry)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.KeyMap.Up):
			currTuple := m.cursor / 2
			isLo := m.cursor % 2
			isHi := (1 - isLo)
			number := m.entry[currTuple*2]*10 + m.entry[currTuple*2+1] + (isHi*10 + isLo)
			if number <= m.maxnum[currTuple] && m.entry[m.cursor] < 9 {
				m.entry[m.cursor]++
			}
		case key.Matches(msg, m.KeyMap.Down):
			currTuple := m.cursor / 2
			isLo := m.cursor % 2
			isHi := (1 - isLo)
			number := m.entry[currTuple*2]*10 + m.entry[currTuple*2+1] - (isHi*10 + isLo)
			if number >= 0 && m.entry[m.cursor] > 0 {
				m.entry[m.cursor]--
			}
		case key.Matches(msg, m.KeyMap.Backspace):
			// if current tuple > 0, zero it
			// if current tuple == 0, zero the previous tuple and jump on it.
			t := m.cursor / 2
			if tupleVal(m.entry[:], t) == 0 && t > 0 {
				t-- // current is 0, so go back one
			}
			m.cursor = t * 2
			m.entry[t*2] = 0
			m.entry[t*2+1] = 0

		case key.Matches(msg, m.KeyMap.Delete):
			m.entry[m.cursor] = 0
		case digitsRe.MatchString(msg.String()):
			whatIf := make([]int, len(m.entry))
			copy(whatIf, m.entry[:])
			v := int(msg.String()[0] - '0')
			whatIf[m.cursor] = v
			currTuple := m.cursor / 2
			if nval := tupleVal(whatIf, currTuple); nval <= m.maxnum[currTuple] {
				m.entry[m.cursor] = v
				if m.cursor < len(m.entry)-1 {
					m.cursor++
				}
			} else if m.cursor%2 == 0 && v*10 <= m.maxnum[currTuple] {
				// if the first digit is legit and we overflow because
				// of the second digit, we zero the second digit and
				// accept the first one.
				m.entry[m.cursor] = v
				m.entry[m.cursor+1] = 0
				m.cursor++
			}
		case key.Matches(msg, m.KeyMap.FocusPrev):
			t := m.cursor / 2
			if t > 0 {
				t--
			}
			m.cursor = t * 2
		case key.Matches(msg, m.KeyMap.FocusNext):
			t := m.cursor / 2
			if t < len(m.entry)/2-1 {
				t++
			}
			m.cursor = t * 2
		}
	}
	return m, nil
}

func (m *Model) updateTime() {
	hour := tupleVal(m.entry[:], 0)
	minute := tupleVal(m.entry[:], 1)
	second := tupleVal(m.entry[:], 2)
	m.Time = time.Date(m.Time.Year(), m.Time.Month(), m.Time.Day(), hour, minute, second, 0, time.UTC)
}

func (m *Model) Value() time.Time {
	m.updateTime()
	return m.Time
}

func (m *Model) toEntry() {
	hour := m.Time.Hour()
	minute := m.Time.Minute()
	second := m.Time.Second()
	m.entry[0] = hour / 10
	m.entry[1] = hour % 10
	m.entry[2] = minute / 10
	m.entry[3] = minute % 10
	m.entry[4] = second / 10
	m.entry[5] = second % 10
}

func tupleVal(entry []int, tuple int) int {
	if len(entry) < tuple*2+1 {
		return -1
	}
	return entry[tuple*2]*10 + entry[tuple*2+1]
}

func (m *Model) View() string {
	if m.finishing {
		return ""
	}

	var (
		r = func(i int) string {
			if i < 0 || len(m.entry) <= i {
				return "?"
			}
			s := strconv.Itoa(m.entry[i])
			if i == m.cursor && m.Focused {
				return m.Styles.Selected.Render(s)
			}
			return m.Styles.Unselected.Render(s)
		}
		sep = m.Styles.Separator.Render(":")
	)

	var buf strings.Builder
	buf.WriteString(drawCursor(m.cursor, 2, '↑', 3) + "\n")
	buf.WriteString(r(0) + r(1) + sep + r(2) + r(3) + sep + r(4) + r(5) + "\n")
	buf.WriteString(drawCursor(m.cursor, 2, '↓', 3))
	if m.ShowHelp {
		buf.WriteString("\n\n" + m.Styles.Help.Render(
			"↓/↑ change, tab jump, backspace zero, delete clear, enter to finish",
		))
	}

	return buf.String()
}

// numTuples is the size of the field in tuples.
func drawCursor(pos int, tupleSz int, char rune, numTuples int) string {
	var buf strings.Builder
	const fill = " "

	before := pos + (pos / tupleSz)
	after := numTuples*tupleSz - before + 1

	buf.WriteString(strings.Repeat(fill, before) + string(char) + strings.Repeat(fill, after))
	return buf.String()
}
