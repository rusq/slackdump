package updaters

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	datepicker "github.com/ethanefung/bubble-datepicker"
)

type DateModel struct {
	Value       *time.Time
	dm          datepicker.Model
	finishing   bool
	timeEnabled bool
}

func NewDTTM(ptrTime *time.Time) DateModel {
	m := datepicker.New(*ptrTime)
	m.SelectDate()
	return DateModel{
		Value:       ptrTime,
		dm:          m,
		timeEnabled: true,
	}
}

func (m DateModel) Init() tea.Cmd {
	return m.dm.Init()
}

func (m DateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			return m, OnClose
		case "enter":
			*m.Value = m.dm.Time
			m.finishing = true
			return m, OnClose
		}
	}

	m.dm, cmd = m.dm.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m DateModel) View() string {
	var b strings.Builder
	b.WriteString(m.dm.View())
	if m.timeEnabled {
		b.WriteString("\n\nTime:  " + m.Value.Format("15:04:05") + " (UTC)")
	}
	b.WriteString("\n\n" + m.dm.Styles.Text.Render("Use arrow keys to navigate, tab/shift+tab to switch between fields, and enter to select."))
	return b.String()
}

// KeyMap is the key bindings for different actions within the datepicker.
type KeyMap struct {
	Up        key.Binding
	Right     key.Binding
	Down      key.Binding
	Left      key.Binding
	FocusPrev key.Binding
	FocusNext key.Binding
	Quit      key.Binding
}

type TimeModel struct {
	t      time.Time
	entry  [6]int
	maxnum [3]int
	cursor int

	KeyMap KeyMap

	focused   bool
	finishing bool
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k", "+")),
		Right:     key.NewBinding(key.WithKeys("right", "l")),
		Down:      key.NewBinding(key.WithKeys("down", "j", "-")),
		Left:      key.NewBinding(key.WithKeys("left", "h")),
		FocusPrev: key.NewBinding(key.WithKeys("shift+tab")),
		FocusNext: key.NewBinding(key.WithKeys("tab")),
		Quit:      key.NewBinding(key.WithKeys("ctrl+c", "q")),
	}
}

func NewTime(t time.Time) *TimeModel {
	return &TimeModel{
		t:       t,
		entry:   [6]int{0, 0, 0, 0, 0, 0},
		maxnum:  [3]int{23, 59, 59},
		cursor:  0,
		focused: false,
		KeyMap:  DefaultKeyMap(),
	}
}

func (m *TimeModel) Focus() {
	m.focused = true
}

func (m *TimeModel) Init() tea.Cmd {
	return nil
}

var digitsRe = regexp.MustCompile(`\d`)

func (m *TimeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Quit):
			m.finishing = true
			return m, tea.Quit
		case key.Matches(msg, m.KeyMap.Left):
			m.cursor--
			if m.cursor < 0 {
				m.cursor = 0
			}
		case key.Matches(msg, m.KeyMap.Right):
			m.cursor++
			if m.cursor > len(m.entry)-1 {
				m.cursor = len(m.entry) - 1
			}
		case key.Matches(msg, m.KeyMap.Up):
			currTuple := m.cursor / 2
			isLo := m.cursor % 2
			isHi := (1 - isLo)
			number := m.entry[currTuple*2]*10 + m.entry[currTuple*2+1] + (isHi*10 + isLo)
			log.Printf("+ number: %d, maxnum: %d, cursor: %d, entry@cursor: %d, hi: %d, lo: %d", number, m.maxnum[currTuple], m.cursor, m.entry[m.cursor], isHi, isLo)
			if number <= m.maxnum[currTuple] && m.entry[m.cursor] < 9 {
				m.entry[m.cursor]++
			}
		case key.Matches(msg, m.KeyMap.Down):
			currTuple := m.cursor / 2
			isLo := m.cursor % 2
			isHi := (1 - isLo)
			number := m.entry[currTuple*2]*10 + m.entry[currTuple*2+1] - (isHi*10 + isLo)
			log.Printf("- number: %d, maxnum: %d, cursor: %d, entry@cursor: %d, hi: %d, lo: %d", number, m.maxnum[currTuple], m.cursor, m.entry[m.cursor], isHi, isLo)
			if number >= 0 && m.entry[m.cursor] > 0 {
				m.entry[m.cursor]--
			}
		case digitsRe.MatchString(msg.String()):
			// TODO: validation
			future := make([]int, 6)
			copy(future, m.entry[:])
			future[m.cursor] = int(msg.String()[0] - '0')
			if tupleVal(future, m.cursor/2) <= m.maxnum[m.cursor/2] {
				m.entry[m.cursor] = int(msg.String()[0] - '0')
			}
			if m.cursor < len(m.entry)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func tupleVal(entry []int, tuple int) int {
	if len(entry) < tuple*2+1 {
		return -1
	}
	return entry[tuple*2]*10 + entry[tuple*2+1]
}

func (m *TimeModel) View() string {
	if m.finishing {
		return ""
	}
	var buf strings.Builder
	buf.WriteString(cursor(m.cursor, 2, '+') + "\n")
	fmt.Fprintf(&buf, "%d%d:%d%d:%d%d\n", m.entry[0], m.entry[1], m.entry[2], m.entry[3], m.entry[4], m.entry[5])
	buf.WriteString(cursor(m.cursor, 2, '-'))

	return buf.String()
}

func cursor(pos int, tupleSz int, char rune) string {
	var buf strings.Builder
	numTuples := pos / tupleSz
	offset := pos % tupleSz

	for i := 0; i < numTuples; i++ {
		buf.WriteString(strings.Repeat(" ", tupleSz) + " ")
	}
	buf.WriteString(strings.Repeat(" ", offset) + string(char))
	return buf.String()
}
