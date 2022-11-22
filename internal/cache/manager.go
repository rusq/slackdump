package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/internal/encio"
)

// Manager is the workspace manager.
type Manager struct {
	dir         string
	authOptions []auth.Option
}

const (
	wspExt         = ".bin"
	defCredsFile   = "provider" + wspExt // default creds file
	defName        = "default"           // name that will be shown for "provider.bin"
	currentWspFile = "workspace.txt"
	userPrefix     = "users-"
	userSuffix     = ".cache"
)

var (
	ErrNoWorkspaces = errors.New("no saved workspaces")
	ErrNameRequired = errors.New("workspace name is required")
	ErrNoDefault    = errors.New("default workspace not set")
)

type Option func(m *Manager)

func WithAuthOpts(opts ...auth.Option) Option {
	return func(m *Manager) {
		m.authOptions = opts
	}
}

// NewManager creates a new workspace manager over the directory dir.
// The cache directory is created with rwx------ permissions, if it does
// not exist.
func NewManager(dir string, opts ...Option) (*Manager, error) {
	m := &Manager{dir: dir}
	for _, opt := range opts {
		opt(m)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return m, nil
}

// Auth authenticates in the Slack Workspace "name" and saves credentials to the
// relevant file. It initialises the auth.Provider depending on provided slack
// credentials. It returns auth.Provider or an error. The logic diagram is
// available in the doc/diagrams/auth_flow.puml.
//
// If the creds is empty, it attempts to load the stored credentials.  If it
// finds them, it returns an initialised credentials provider.  If not - it
// returns the auth provider according to the type of credentials determined
// by creds.AuthProvider, and saves them to an AES-256-CFB encrypted storage.
//
// The storage is encrypted using the hash of the unique machine-ID, supplied by
// the operating system (see package encio), it makes it impossible use the
// stored credentials on another machine (including virtual), even another
// operating system on the same machine, unless it's a clone of the source
// operating system on which the credentials storage was created.
func (m *Manager) Auth(ctx context.Context, name string, c Credentials) (auth.Provider, error) {
	return initProvider(ctx, m.dir, m.filename(name), name, c, m.authOptions...)
}

type ErrWorkspace struct {
	Workspace string
	Message   string
	Err       error
}

func (ew *ErrWorkspace) Error() string {
	if ew.Err == nil {
		return fmt.Sprintf("workspace %q: %s", ew.Workspace, ew.Message)
	}
	return fmt.Sprintf("workspace %q: %s (error: %s)", ew.Workspace, ew.Message, ew.Err)
}

func newErrNoWorkspace(name string) *ErrWorkspace {
	return &ErrWorkspace{Workspace: name, Message: "no such workspace"}
}

func (ew *ErrWorkspace) Unwrap() error {
	return ew.Err
}

// Delete deletes the workspace file.
func (m *Manager) Delete(name string) error {
	if !m.Exists(name) {
		return newErrNoWorkspace(name)
	}
	if err := os.Remove(m.filepath(name)); err != nil {
		return &ErrWorkspace{Workspace: name, Message: "failed to delete", Err: err}
	}
	return nil
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
		return nil, fmt.Errorf("error listing existing workspaces: %w", err)
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
		return m.selectDefault()
	}
	defer f.Close()
	wf := m.readWsp(f)

	if !exist(workspaces, wf) {
		return m.selectDefault()
	}

	return wf, nil
}

func (m *Manager) selectDefault() (string, error) {
	if !m.Exists(defName) {
		return "", ErrNoDefault
	}
	if err := m.Select(defName); err != nil {
		return "", err
	}
	return defName, nil
}

// Select selects the existing workspace with "name"
func (m *Manager) Select(name string) error {
	if !m.Exists(name) {
		return newErrNoWorkspace(name)
	}

	f, err := os.Create(filepath.Join(m.dir, currentWspFile))
	if err != nil {
		return &ErrWorkspace{Workspace: name, Message: "failed to create workspace file", Err: err}
	}
	defer f.Close()
	return m.writeWsp(f, name)
}

// FileInfo returns the container file information for the workspace.
func (m *Manager) FileInfo(name string) (fs.FileInfo, error) {
	fi, err := os.Stat(m.filepath(name))
	if err != nil {
		return nil, &ErrWorkspace{Workspace: name, Message: "error accessing workspace file", Err: err}
	}
	return fi, nil
}

// Exists returns true if the workspace with name "name" exists in the list of
// authenticated workspaces.
func (m *Manager) Exists(name string) bool {
	existing, err := m.List()
	if err != nil {
		return false
	}
	return exist(existing, name)
}

// filename returns the filename for the workspace name.
func (m *Manager) filename(name string) string {
	if name == defName || name == "" {
		name = defCredsFile
	} else {
		name = name + wspExt
	}
	return name
}

// filepath returns the full path to the filename of workspace name.
func (m *Manager) filepath(name string) string {
	return filepath.Join(m.dir, m.filename(name))
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
		return defCredsFile
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

// WalkUsers scans the cache directory and calls userFn for each user file
// discovered.
func (m *Manager) WalkUsers(userFn func(path string, r io.Reader) error) error {
	err := filepath.WalkDir(m.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasPrefix(d.Name(), userPrefix) && !strings.HasSuffix(d.Name(), userSuffix) {
			// skip non-matching files
			return nil
		}
		f, err := encio.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		return userFn(path, f)
	})
	return err
}
