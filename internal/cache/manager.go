package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/rusq/encio"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/auth"
)

// Manager is the workspace manager.  It is an abstraction over the directory
// with files with credentials for Slack workspaces.
//
// There are several types of files that one could find in the managed
// directory:
//   - "provider.bin" - the default workspace file, it contains the credentials
//     for the "default" workspace.  It exists for historical reasons, for
//     migration from the previous version of the slackdump.  It contains
//     credentials for the workspace that slackdump v2 was authenticated in.
//   - "*.bin" - other workspaces, the filename is the name of the workspace.
//   - "workspace.txt" - a pointer to the current workspace, it contains the
//     current workspace name.
//   - "*.cache" - cache files, they contain the cache for users and channels.
type Manager struct {
	dir         string
	authOptions []auth.Option

	userFile    string
	channelFile string
}

const (
	wspExt         = ".bin"              // workspace file extension
	defCredsFile   = "provider" + wspExt // default creds file
	defName        = "default"           // name that will be shown for "provider.bin"
	currentWspFile = "workspace.txt"
)

var (
	ErrNoWorkspaces = errors.New("no saved workspaces")
	ErrNameRequired = errors.New("workspace name is required")
	ErrNoDefault    = errors.New("default workspace not set")
)

type Option func(m *Manager)

// WithAuthOpts allows to change the default Auth options, they will be
// passed to auth package..
func WithAuthOpts(opts ...auth.Option) Option {
	return func(m *Manager) {
		m.authOptions = opts
	}
}

// WithChannelCacheBase allows to change the default cache file name for
// channels cache.
func WithChannelCacheBase(filename string) Option {
	return func(m *Manager) {
		if filename == "" {
			return
		}
		m.channelFile = maybeAppendExt(filename, ".cache")
	}
}

// WithUserCacheBase allows to change the default base name of "users.cache".
// If the filename is empty it's a noop.  If the filename does not contain
// extension, ".cache" is appended.
func WithUserCacheBase(filename string) Option {
	return func(m *Manager) {
		if filename == "" {
			return
		}
		m.userFile = maybeAppendExt(filename, ".cache")
	}
}

// maybeAppendExt appends the extension to the filename if it's empty.
func maybeAppendExt(filename string, ext string) string {
	if ext == "" {
		return filename
	}
	if ext := filepath.Ext(filename); ext == "" || ext == "." {
		filename += ext
	}
	return filename
}

// NewManager creates a new workspace manager over the directory dir.  The
// directory is created with rwx------ permissions, if it does not exist.
//
// TODO: test with empty dir.
func NewManager(dir string, opts ...Option) (*Manager, error) {
	m := &Manager{
		dir:         dir,
		userFile:    "users.cache",
		channelFile: "channels.cache",
	}
	for _, opt := range opts {
		opt(m)
	}
	if m.dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
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

// LoadProvider loads the file from disk without any checks.
func (m *Manager) LoadProvider(name string) (auth.Provider, error) {
	return loadCreds(filer, filepath.Join(m.dir, m.filename(name)))
}

// ErrWorkspace is the error returned by the workspace manager, it contains the
// workspace name, the error message and the underlying error.
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

// Unwrap returns the underlying error.
func (ew *ErrWorkspace) Unwrap() error {
	return ew.Err
}

func newErrNoWorkspace(name string) *ErrWorkspace {
	return &ErrWorkspace{Workspace: name, Message: "no such workspace"}
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

	if !slices.Contains(workspaces, wf) {
		return m.selectDefault()
	}

	return wf, nil
}

// selectDefault selects the default workspace if it exists.
func (m *Manager) selectDefault() (string, error) {
	var wsp = defName
	if !m.HasDefault() {
		// the default workspace does not exist, pick any
		w, err := m.firstAvailable()
		if err != nil {
			return "", err
		}
		wsp = w
	}
	if err := m.Select(wsp); err != nil {
		return "", err
	}
	return wsp, nil
}

func (m *Manager) HasDefault() bool {
	return m.Exists(defName)
}

func (m *Manager) firstAvailable() (string, error) {
	all, err := m.List()
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "", ErrNoWorkspaces
	}
	return all[0], nil
}

// Select selects the existing workspace with "name".
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
	return slices.Contains(existing, name)
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

// name returns the workspace name from the filename.
func (m *Manager) name(filename string) (string, error) {
	if filedir := filepath.Dir(filename); !strings.EqualFold(filedir, m.dir) {
		return "", fmt.Errorf("incorrect directory: %s", filedir)
	}
	if filepath.Ext(filename) != wspExt {
		return "", fmt.Errorf("invalid workspace extension: %s", filepath.Ext(filename))
	}
	return wspName(filename), nil
}

// readWsp reads the workspace file name from the reader.
func (m *Manager) readWsp(r io.Reader) string {
	var current string
	if _, err := fmt.Fscanln(r, &current); err != nil {
		return defCredsFile
	}
	return strings.TrimSpace(current)
}

// writeWsp writes the workspace file name to writer w.
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

// WalkUsers scans the cache directory and calls userFn for each user file
// discovered.
func (m *Manager) WalkUsers(userFn func(path string, r io.Reader) error) error {
	userSuffix := filepath.Ext(m.userFile)
	userPrefix := m.userFile[0 : len(m.userFile)-len(userSuffix)]
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

// LoadUsers loads user cache file no older than maxAge for teamID.
func (m *Manager) LoadUsers(teamID string, maxAge time.Duration) ([]slack.User, error) {
	return loadUsers(m.dir, m.userFile, teamID, maxAge)
}

// CacheUsers saves users to user cache file for teamID.
func (m *Manager) CacheUsers(teamID string, uu []slack.User) error {
	return saveUsers(m.dir, m.userFile, teamID, uu)
}

// LoadChannels loads channel cache no older than maxAge.
func (m *Manager) LoadChannels(teamID string, maxAge time.Duration) ([]slack.Channel, error) {
	return loadChannels(m.dir, m.channelFile, teamID, maxAge)
}

// CacheChannels saves channels to cache.
func (m *Manager) CacheChannels(teamID string, cc []slack.Channel) error {
	return saveChannels(m.dir, m.channelFile, teamID, cc)
}
