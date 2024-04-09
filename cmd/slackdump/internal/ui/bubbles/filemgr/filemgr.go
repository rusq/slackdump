package filemgr

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	Globs     []string
	Selected  string
	Directory string
	Height    int
	ShowHelp  bool
	Style     Style
	files     []fs.FileInfo
	finished  bool
	st        state
	viewStack stack[state]

	Debug bool
	last  string // last key pressed
}

type Style struct {
	Normal    lipgloss.Style
	Directory lipgloss.Style
	Inverted  lipgloss.Style
}

func NewModel(dir string, height int, globs ...string) Model {
	return Model{
		Globs:     globs,
		Directory: dir,
		Height:    height,
		Style: Style{
			Normal:    lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
			Directory: lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
			Inverted:  lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Background(lipgloss.Color("240")),
		},
	}
}

type state struct {
	cursor   int
	max, min int
}

type stack[T any] []T

func (s *stack[T]) Push(v T) {
	*s = append(*s, v)
}

func (s *stack[T]) Pop() T {
	var empty T
	if len(*s) == 0 {
		return empty
	}
	v := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return v
}

func (s stack[T]) Len() int {
	return len(s)
}

func (s stack[T]) Peek() T {
	return s[len(s)-1]
}

type wmReadDir struct {
	dir   string
	files []fs.FileInfo
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		dirs, err := collectDirs(m.Directory)
		if err != nil {
			return err
		}
		files, err := collectFiles(m.Directory, m.Globs...)
		if err != nil {
			return err
		}
		return wmReadDir{m.Directory, append(files, dirs...)}
	}
}

func collectFiles(dir string, globs ...string) (files []fs.FileInfo, err error) {
	err = fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != "." {
			return fs.SkipDir
		}
		for _, glob := range globs {
			if ok, err := filepath.Match(glob, d.Name()); err != nil {
				return err
			} else if ok {
				fi, err := d.Info()
				if err != nil {
					return err
				}
				files = append(files, fi)
			}
		}
		return nil
	})
	return
}

func collectDirs(dir string) ([]fs.FileInfo, error) {
	var dirs []fs.FileInfo
	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == "." {
				return nil
			}
			dir, err := d.Info()
			if err != nil {
				return err
			}
			dirs = append(dirs, dir)
			return fs.SkipDir
		}
		return nil
	})
	return dirs, err
}

func (m Model) height() int {
	if m.ShowHelp {
		return m.Height - 2
	}
	return m.Height
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case error:
		log.Printf("error: %v", msg)
		return m, tea.Quit
	case tea.KeyMsg:
		m.last = msg.String()
		switch msg.String() {
		case "ctrl+c", "q":
			m.finished = true
			return m, tea.Quit
		case "up", "ctrl+p", "k":
			if m.st.cursor > 0 {
				m.st.cursor--
			}
			if m.st.cursor < m.st.min {
				m.st.min--
				m.st.max--
			}
		case "down", "ctrl+n", "j":
			if m.st.cursor < len(m.files)-1 {
				m.st.cursor++
			}
			if m.st.cursor > m.st.max {
				m.st.min++
				m.st.max++
			}
		case "right", "pgdown", "ctrl+v", "ctrl+f":
			m.st.cursor += m.height()
			if m.st.cursor > len(m.files)-1 {
				m.st.cursor = len(m.files) - 1
			}
			m.st.min += m.height()
			m.st.max += m.height()
			if m.st.max >= len(m.files) {
				m.st.max = len(m.files) - 1
				m.st.min = m.st.max - (m.height() - 1)
			}
		case "left", "pgup", "alt+v", "ctrl+b":
			m.st.cursor -= m.height()
			if m.st.cursor < 0 {
				m.st.cursor = 0
			}
			m.st.min -= m.height()
			m.st.max -= m.height()
			if m.st.min < 0 {
				m.st.min = 0
				m.st.max = m.height() - 1
			}
		case "ctrl+r":
			return m, tea.Batch(m.Init())
		case "enter", "ctrl+m":
			if len(m.files) == 0 {
				break
			}
			if m.files[m.st.cursor].IsDir() {
				m.Directory = filepath.Join(m.Directory, m.files[m.st.cursor].Name())
				m.viewStack.Push(m.st)
				m.st = state{}
				return m, tea.Batch(m.Init())
			}
			cmds = append(cmds, selectedCmd(m.Directory, m.files[m.st.cursor]))
		case "backspace", "ctrl+h":
			if m.viewStack.Len() > 0 {
				m.st = m.viewStack.Pop()
				m.Directory = filepath.Dir(m.Directory)
				return m, tea.Batch(m.Init())
			}
		}
	case wmReadDir:
		m.files = msg.files
		m.st.max = max(m.st.max, m.height()-1)
	}

	return m, tea.Batch(cmds...)
}

func selectedCmd(dir string, fi fs.FileInfo) tea.Cmd {
	return func() tea.Msg {
		return WMSelected{
			Filepath: filepath.Join(dir, fi.Name()),
			IsDir:    fi.IsDir(),
		}
	}
}

type WMSelected struct {
	Filepath string
	IsDir    bool
}

// humanizeSize returns a human-readable string representing a file size.
// for example 240.4M or 2.3G
func humanizeSize(size int64) string {
	const (
		K = 1 << 10
		M = 1 << 20
		G = 1 << 30
		T = 1 << 40
	)

	switch {
	case size < K:
		return fmt.Sprintf("%5dB", size)
	case size < M:
		return fmt.Sprintf("%5.1fK", float64(size)/K)
	case size < G:
		return fmt.Sprintf("%5.1fM", float64(size)/M)
	case size < T:
		return fmt.Sprintf("%5.1fG", float64(size)/G)
	default:
		return fmt.Sprintf("%5.1fT", float64(size)/G)

	}
}

const Width = 40

func printFile(fi fs.FileInfo) string {
	// filename.extension  <DIR>  02-01-2006 15:04
	const (
		dttmLayout = "02-01-2006 15:04"
		dirMarker  = "<DIR>"
		filesizeSz = 6
		dttmSz     = len(dttmLayout)
		filenameSz = Width - filesizeSz - dttmSz - 3
	)

	var sz = dirMarker
	if !fi.IsDir() {
		sz = humanizeSize(fi.Size())
	}
	return fmt.Sprintf("%-*s %*s %s", filenameSz, trunc(fi.Name(), filenameSz), filesizeSz, sz, fi.ModTime().Format(dttmLayout))
}

func trunc(s string, sz int) string {
	if len(s) > sz {
		return s[:sz-1] + "…"
	}
	return s
}

func (m Model) View() string {
	if m.finished {
		return ""
	}
	var buf strings.Builder
	if m.Debug {
		fmt.Fprintf(&buf, "cursor: %d\n", m.st.cursor)
		fmt.Fprintf(&buf, "min: %d\n", m.st.min)
		fmt.Fprintf(&buf, "max: %d\n", m.st.max)
		fmt.Fprintf(&buf, "last: %q\n", m.last)
		fmt.Fprintf(&buf, "dir: %q\n", m.Directory)
		fmt.Fprintf(&buf, "selected: %q\n", m.Selected)
		for i := range Width {
			if i%10 == 0 {
				buf.WriteByte('|')
			} else {
				fmt.Fprint(&buf, i%10)
			}
		}
		fmt.Fprintln(&buf)
	}
	if len(m.files) == 0 {
		buf.WriteString(m.Style.Normal.Render("No files found, press [Backspace]") + "\n")
		for i := 0; i < m.height()-1; i++ {
			fmt.Fprintln(&buf, m.Style.Normal.Render(strings.Repeat(" ", Width-1))) //padding
		}
	} else {
		for i, file := range m.files {
			if i < m.st.min || i > m.st.max {
				continue
			}
			style := m.Style.Normal
			if file.IsDir() {
				style = m.Style.Directory
			}
			if i == m.st.cursor {
				style = m.Style.Inverted
			}
			fmt.Fprintln(&buf, style.Render(printFile(file)))
		}
		numDisplayed := min(len(m.files), m.st.max-m.st.min+1)
		for i := 0; i < m.height()-numDisplayed; i++ {
			fmt.Fprintln(&buf, m.Style.Normal.Render(strings.Repeat(" ", Width-1)))
		}
	}
	if m.ShowHelp {
		buf.WriteString("\n ↑ ↓ move・[⏎] select・[⇤] back・[q] quit\n")
	}
	return buf.String()
}
