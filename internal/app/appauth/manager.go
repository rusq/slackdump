package appauth

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Manager is the workspace manager.
type Manager struct {
	dir string
}

const (
	wspExt         = ".bin"
	defCredsFile   = "provider" + wspExt // default creds file
	defName        = "default"           // name that will be shown for "provider.bin"
	currentWspFile = "workspace.txt"
)

var ErrNoWorkspaces = errors.New("no saved workspaces")

// NewManager creates a new workspace manager over the directory dir.
// The cache directory is created with rwx------ permissions, if it does
// not exist.
func NewManager(dir string) (*Manager, error) {
	m := &Manager{dir: dir}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) List() ([]string, error) {
	files, err := m.listFiles()
	if err != nil {
		return nil, err
	}
	var workspaces = make([]string, len(files))
	for i := range files {
		name, err := m.name(files[i])
		if err != nil {
			return nil, fmt.Errorf("internal error: %s", err)
		}
		workspaces[i] = name
	}
	return workspaces, nil
}

// List returns the list of workspace files with full path.
func (m *Manager) listFiles() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(m.dir, "*"+wspExt))
	if err != nil {
		return nil, fmt.Errorf("error trying to find existing workspaces: %w", err)
	}
	if len(files) == 0 {
		return nil, ErrNoWorkspaces
	}
	sort.Strings(files)
	return files, nil
}

// Current returns the current workspace name.
func (m *Manager) Current() (string, error) {
	workspaces, err := m.List()
	if err != nil {
		return "", err
	}

	f, err := os.Open(filepath.Join(m.dir, currentWspFile))
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		// if workspace file does not exist, select the default one.
		if !exist(workspaces, defName) {
			return "", err
		}
		if e := m.Select(defName); e != nil {
			return "", e
		}
		return defName, nil
	}
	defer f.Close()
	wf := m.readWsp(f)

	if !exist(workspaces, wf) {
		return "", fmt.Errorf("unknown workspace %s", wf)
	}

	return wf, nil
}

// Select selects the existing workspace with "name"
func (m *Manager) Select(name string) error {
	existing, err := m.List()
	if err != nil {
		return err
	}
	if !exist(existing, name) {
		return fmt.Errorf("unknown workspace %s", name)
	}

	f, err := os.Create(filepath.Join(m.dir, currentWspFile))
	if err != nil {
		return err
	}
	defer f.Close()
	return m.writeWsp(f, name)
}

// FileInfo returns the container file information for the workspace.
func (m *Manager) FileInfo(name string) (fs.FileInfo, error) {
	fi, err := os.Stat(m.filename(name))
	if err != nil {
		return nil, err
	}
	return fi, nil
}

// filename returns the full path to the filename of workspace name.
func (m *Manager) filename(name string) string {
	if name == defName {
		name = defCredsFile
	} else {
		name = name + wspExt
	}
	return filepath.Join(m.dir, name)
}

func (m *Manager) name(filename string) (string, error) {
	if filedir := filepath.Dir(filename); !strings.EqualFold(filedir, m.dir) {
		return "", fmt.Errorf("incorrect directory: %s", filedir)
	}
	if filepath.Ext(filename) != wspExt {
		return "", fmt.Errorf("invalid workspace extension: %s", filepath.Ext(filename))
	}
	return wspName(filename), nil
}

func (m *Manager) readWsp(r io.Reader) string {
	var current string
	if _, err := fmt.Fscanln(r, &current); err != nil {
		return filepath.Join(m.dir, defCredsFile)
	}
	return strings.TrimSpace(current)
}

func (*Manager) writeWsp(w io.Writer, filename string) error {
	_, err := fmt.Fprintln(w, filename)
	return err
}

// wspName returns the workspace name for the file.
func wspName(filename string) string {
	name := filepath.Base(filename)
	if name == defCredsFile {
		name = defName
	} else {
		ext := filepath.Ext(name)
		name = name[:len(name)-len(ext)]
	}
	return name
}

func indexOf[T comparable](ss []T, s T) int {
	for i := range ss {
		if s == ss[i] {
			return i
		}
	}
	return -1
}

func exist[T comparable](ss []T, s T) bool {
	return -1 < indexOf(ss, s)
}
