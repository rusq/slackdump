package filemgr

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rusq/rbubbles/display"
)

type Model struct {
	Globs      []string
	Selected   string
	FS         fs.FS
	BaseDir    string
	Directory  string
	Height     int
	ShowHelp   bool
	ShowCurDir bool
	Style      Style
	files      []fs.FileInfo
	finished   bool
	focus      bool
	st         display.State
	viewStack  display.Stack[display.State]

	Debug bool
	last  string // last key pressed
}

type Style struct {
	Normal    lipgloss.Style
	Directory lipgloss.Style
	Inverted  lipgloss.Style
	Shaded    lipgloss.Style
	CurDir    lipgloss.Style
}

// Messages
type (
	// WMSelected message is sent by the file manager when a file is selected.
	WMSelected struct {
		Filepath string
		IsDir    bool
	}

	wmReadDir struct {
		dir   string
		files []fs.FileInfo
	}
)

// New creates a new file manager model over the filesystem fsys.  The base
// directory is what will be displayed in the file manager.  The dir is the
// current directory within the fsys.  The height is the number of lines to
// display.  The globs are the file globs to display.
func New(fsys fs.FS, base string, dir string, height int, globs ...string) Model {
	return Model{
		Globs:      globs,
		FS:         fsys,
		Directory:  dir,
		BaseDir:    base,
		Height:     height,
		focus:      false,
		ShowCurDir: true,
		// Sensible defaults
		Style: Style{
			Normal:    lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
			Directory: lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
			Inverted:  lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Background(lipgloss.Color("240")),
			CurDir:    lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
		},
	}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		slog.Debug("init", "dir", m.Directory, "globs", m.Globs)
		msg, err := readFS(m.FS, m.Directory, m.Globs...)
		if err != nil {
			slog.Error("readFS", "err", err)
			return err
		}
		slog.Debug("readFS", "msg", msg)
		return msg
	}
}

func readFS(fsys fs.FS, dir string, globs ...string) (wmReadDir, error) {
	sub, err := fs.Sub(fsys, filepath.Clean(dir))
	if err != nil {
		return wmReadDir{}, fmt.Errorf("sub: %w", err)
	}
	dirs, err := collectDirs(sub)
	if err != nil {
		return wmReadDir{}, fmt.Errorf("collect dirs: %w", err)
	}
	files, err := collectFiles(sub, globs...)
	if err != nil {
		return wmReadDir{}, fmt.Errorf("collectFiles: %w", err)
	}
	if dir != "." && dir != string(filepath.Separator) && dir != "" {
		files = append([]fs.FileInfo{specialDir{".."}}, files...)
	}
	return wmReadDir{dir, append(files, dirs...)}, nil
}

func collectFiles(fsys fs.FS, globs ...string) (files []fs.FileInfo, err error) {
	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			// do not show the current directory
			return nil
		}
		if d.IsDir() {
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

func collectDirs(fsys fs.FS) ([]fs.FileInfo, error) {
	var dirs []fs.FileInfo
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		if d.IsDir() {
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

func (m *Model) populate(files []fs.FileInfo) {
	m.files = files
	m.st.SetMax(m.height())
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	// we only care about wmReadDir messages if we're not focused.
	switch msg := msg.(type) {
	case wmReadDir:
		slog.Debug("wmReadDir", "dir", msg.dir)
		m.populate(msg.files)
	}

	if !m.focus {
		return m, nil
	}

	var cmds []tea.Cmd
	slog.Debug("filemanager.Update", "msg", msg)
	switch msg := msg.(type) {
	case error:
		slog.Error("error message", "msg", msg)
	case tea.WindowSizeMsg:
		if m.Height == 0 {
			m.Height = msg.Height
		}
	case tea.KeyMsg:
		if !m.focus {
			break
		}
		m.last = msg.String()
		switch msg.String() {
		case "up", "ctrl+p", "k":
			m.st.Up()
		case "down", "ctrl+n", "j":
			m.st.Down(len(m.files))
		case "right", "pgdown", "ctrl+v", "ctrl+f":
			m.st.NextPg(m.height(), len(m.files))
		case "left", "pgup", "alt+v", "ctrl+b":
			m.st.PrevPg(m.height())
		case "home":
			m.st.Home(m.height())
		case "end":
			m.st.End(m.height(), len(m.files))
		case "ctrl+r":
			return m, tea.Batch(m.Init())
		case "enter", "ctrl+m":
			if len(m.files) == 0 {
				break
			}
			if m.files[m.st.Cursor].IsDir() {
				m.Directory = filepath.Join(m.Directory, m.files[m.st.Cursor].Name())
				m.viewStack.Push(m.st)
				m.st = display.State{}
				return m, tea.Batch(m.Init())
			}
			cmds = append(cmds, selectedCmd(m.Directory, m.files[m.st.Cursor]))
		case "backspace", "ctrl+h":
			if m.viewStack.Len() > 0 {
				m.st = m.viewStack.Pop()
				m.Directory = filepath.Dir(m.Directory)
				return m, tea.Batch(m.Init())
			}
		}
		if combo := msg.String(); strings.HasPrefix(combo, "alt+") {
			_, key, found := strings.Cut(combo, "+")
			if !found {
				break
			}
			for i, f := range m.files {
				if strings.HasPrefix(strings.ToLower(f.Name()), key) {
					if m.last == combo && i <= m.st.Cursor {
						continue
					}
					m.st.Focus(i, m.height(), len(m.files))
					break
				}
			}
		}

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
		return fmt.Sprintf("%5.1fT", float64(size)/T)

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

	sz := dirMarker
	if !fi.IsDir() {
		sz = humanizeSize(fi.Size())
	}
	return fmt.Sprintf("%-*s %*s %s", filenameSz, display.Trunc(fi.Name(), filenameSz), filesizeSz, sz, fi.ModTime().Format(dttmLayout))
}

func (m Model) printDebug(w io.Writer) {
	fmt.Fprintf(w, "cursor: %d\n", m.st.Cursor)
	fmt.Fprintf(w, "min: %d\n", m.st.Min)
	fmt.Fprintf(w, "max: %d\n", m.st.Max)
	fmt.Fprintf(w, "last: %q\n", m.last)
	fmt.Fprintf(w, "dir: %q\n", m.Directory)
	fmt.Fprintf(w, "selected: %q\n", m.Selected)
	for i := range Width {
		if n := i % 10; n == 0 {
			w.Write([]byte{'|'})
		} else {
			fmt.Fprint(w, n)
		}
	}
	fmt.Fprintln(w)
}

func (m Model) View() string {
	if m.finished {
		return ""
	}
	var buf strings.Builder
	if m.Debug {
		m.printDebug(&buf)
	}
	if m.ShowCurDir {
		buf.WriteString(
			m.Style.CurDir.Render(
				fmt.Sprintf("DIR: %s", m.shorten(filepath.Join(
					filepath.Clean(m.BaseDir),
					filepath.Clean(m.Directory),
				))),
			) + "\n",
		)
	}
	if len(m.files) == 0 {
		buf.WriteString(m.Style.Normal.Render("No files found, press [Backspace]") + "\n")
		for i := 0; i < m.height()-1; i++ {
			fmt.Fprintln(&buf, m.Style.Normal.Render(strings.Repeat(" ", Width-1))) // padding
		}
	} else {
		for i, file := range m.files {
			if i < m.st.Min || i > m.st.Max {
				continue
			}
			style := m.Style.Normal
			if file.IsDir() {
				style = m.Style.Directory
			}
			if i == m.st.Cursor {
				if m.focus {
					style = m.Style.Inverted
				} else {
					style = m.Style.Shaded
				}
			}
			fmt.Fprintln(&buf, style.Render(printFile(file)))
		}
		numDisplayed := m.st.Displayed(len(m.files))
		for i := 0; i < m.height()-numDisplayed; i++ {
			fmt.Fprintln(&buf, m.Style.Normal.Render(strings.Repeat(" ", Width-1)))
		}
	}
	if m.ShowHelp {
		buf.WriteString("\n ↑↓ move•[⏎] select•[⇤] back•[q] quit")
	}
	return buf.String()
}

func (m *Model) Select(filename string) {
	if len(m.files) == 0 {
		w, err := readFS(m.FS, m.Directory, m.Globs...)
		if err != nil {
			slog.Error("readFS", "err", err)
			return
		}
		m.populate(w.files)
	}
	for i, f := range m.files {
		if f.Name() == filename {
			m.st.Focus(i, m.height(), len(m.files))
			break
		}
	}
}

func (m *Model) Focus() {
	m.focus = true
}

func (m *Model) Blur() {
	m.focus = false
}

type specialDir struct {
	name string
}

func (s specialDir) Name() string {
	return s.name
}

func (s specialDir) Size() int64 {
	return 0
}

func (s specialDir) Mode() fs.FileMode {
	return fs.ModeDir
}

func (s specialDir) ModTime() time.Time {
	return time.Time{}
}

func (s specialDir) IsDir() bool {
	return true
}

func (s specialDir) Sys() interface{} {
	return s
}

// shorten returns a shortened version of a path.
func (m Model) shorten(dirpath string) string {
	dirpath = filepath.Clean(dirpath)
	if len(dirpath) < Width-1 {
		return dirpath
	}
	dirpath = filepath.Clean(dirpath)
	// split the path into parts
	parts := strings.Split(dirpath, string(filepath.Separator))
	var s []string
	if len(parts) < 2 {
		return dirpath
	}
	for i := 0; i < len(parts)-1; i++ {
		if len(parts[i]) == 0 {
			s = append(s, "")
			continue
		}
		if strings.HasSuffix(parts[i], ":") {
			s = append(s, parts[i])
			continue
		}
		s = append(s, string(parts[i][0]))
	}
	s = append(s, parts[len(parts)-1])
	if runtime.GOOS == "windows" {
		s[1] = "\\" + s[1]
	}
	res := filepath.Join(s...)
	if dirpath[0] == '/' {
		res = "/" + res
	}
	if len(res) > Width-1 {
		res = "…" + res[len(res)-Width+3:]
	}
	return res
}
