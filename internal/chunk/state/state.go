package state

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3/internal/fasttime"
	"github.com/rusq/slackdump/v3/internal/osext"
)

const Version = 0.1

// these types exist to make the code more readable.
type (
	_id          = string
	_idAndThread = string
)

// ErrStateVersion is returned when the state version does not match the
// expected version.
type ErrStateVersion struct {
	Expected float64
	Actual   float64
}

func (e ErrStateVersion) Error() string {
	return "state version mismatch: expected " + strconv.FormatFloat(e.Expected, 'f', -1, 64) + ", got " + strconv.FormatFloat(e.Actual, 'f', -1, 64)
}

// State is a struct that holds the state of the Slack dump.
type State struct {
	// Version is the version of the state file.
	Version float64 `json:"version"`
	// ChunkFilename is the original chunks filename for which the state is valid.
	// It may be empty.
	ChunkFilename string `json:"chunk_filename,omitempty"`
	// IsComplete indicates that all chunks were written to the file.
	IsComplete bool `json:"is_complete"`
	// Directory with downloaded files, if any.
	FilesDir string `json:"files_dir,omitempty"`
	// IsCompressed indicates that the chunk file is compressed.
	IsCompressed bool `json:"is_compressed,omitempty"`
	// Channels is a map of channel ID to the latest message timestamp.
	Channels map[_id]int64 `json:"channels,omitempty"`
	// Threads is a map of channel ID + thread timestamp to the latest message
	// timestamp in that thread.
	Threads map[_idAndThread]int64 `json:"threads,omitempty"`
	// Files is a map of file ID to the channel ID where it was posted.
	Files map[_id]_id `json:"files,omitempty"`
	// ChannelInfos contains the list of all channels in the state file.
	ChannelInfos []string `json:"channel_infos,omitempty"`

	mu sync.RWMutex
}

// Stater is an interface for types that can return a State.
type Stater interface {
	// State should return the State of the underlying type.
	State() (*State, error)
}

// New returns a new State.
func New(filename string) *State {
	return &State{
		Version:       Version,
		ChunkFilename: filename,
		Channels:      make(map[_id]int64),
		Threads:       make(map[_idAndThread]int64),
		Files:         make(map[_id]_id),
	}
}

// State holds the state of a chunk recording. It contains the filename of the
// chunk recording file, as well as the path to the downloaded files.

var ErrNoChunkFile = errors.New("no linked chunk file")

// OpenChunks attempts to open the chunk file linked in the State. If the
// chunk is compressed, it will be decompressed and a temporary file will be
// created. The temporary file will be removed when the OpenChunks is
// closed.
func (st *State) OpenChunks(basePath string) (io.ReadSeekCloser, error) {
	if st.ChunkFilename == "" {
		return nil, ErrNoChunkFile
	}
	f, err := os.Open(filepath.Join(basePath, st.ChunkFilename))
	if err != nil {
		return nil, err
	}
	if st.IsCompressed {
		tf, err := osext.UnGZIP(f)
		if err != nil {
			return nil, err
		}
		return osext.RemoveOnClose(tf), nil
	}
	return f, nil
}

// AddMessage indexes the message.  It should be called when a message is
// processed.
func (s *State) AddMessage(channelID, messageTS string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Channels == nil {
		s.Channels = make(map[_id]int64)
	}
	tsUpdate(s.Channels, channelID, messageTS)
}

// AddThread adds a thread to index.  It should be called when a thread is
// processed.
func (s *State) AddThread(channelID, threadTS, ts string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Threads == nil {
		s.Threads = make(map[_idAndThread]int64)
	}
	tsUpdate(s.Threads, threadID(channelID, threadTS), ts)
}

// AddFile adds a file to index.  It should be called when a file is processed.
func (s *State) AddFile(channelID, fileID string, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Files == nil {
		s.Files = make(map[_id]_id)
	}
	s.Files[channelID+":"+fileID] = path
}

// AllFiles returns all saved files for the given channel.
func (s *State) AllFiles(channelID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var files []string
	for fileChanID, path := range s.Files {
		id, _, _ := strings.Cut(fileChanID, ":")
		if id == channelID {
			files = append(files, path)
		}
	}
	return files
}

// AddChannel should be called when a channel is processed.
func (s *State) AddChannel(info string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ChannelInfos = append(s.ChannelInfos, info)
}

// FilePath returns the file path for the given file ID in the given channel.
func (s *State) FilePath(channelID, fileID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Files[channelID+":"+fileID]
}

// tsUpdate updates the map with the given ID and value if the value is greater.
func tsUpdate(m map[string]int64, id string, val string) {
	currVal, err := fasttime.TS2int(val)
	if err != nil {
		return // not updating crooked values
	}
	existingVal, ok := m[id]
	if !ok || currVal > existingVal {
		m[id] = currVal
	}
}

// HasChannel returns true if the channel is known (has at least one message).
func (s *State) HasChannel(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return has(s.Channels, id)
}

// HasThread returns true if the thread is known (has at least one message).
func (s *State) HasThread(channelID, threadTS string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return has(s.Threads, threadID(channelID, threadTS))
}

// HasFile returns true if the file is known.
func (s *State) HasFile(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return has(s.Files, id)
}

// has returns true if the map contains the given key
func has[T any](m map[string]T, k string) bool {
	if m == nil {
		return false
	}
	_, ok := m[k]
	return ok
}

// LatestChannelTS returns the latest known message timestamp for the given
// channel.
func (s *State) LatestChannelTS(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return latest(s.Channels, id)
}

// LatestThreadTS returns the latest known message timestamp for the given
// thread.
func (s *State) LatestThreadTS(channelID, threadTS string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return latest(s.Threads, threadID(channelID, threadTS))
}

// threadID returns the ID for the given thread.  The ID is the channel ID
// concatenated with the thread timestamp.
func threadID(channelID, threadTS string) string {
	return channelID + ":" + threadTS
}

// latest returns the latest known timestamp for the given ID.
func latest(m map[string]int64, id string) string {
	if m == nil {
		return ""
	}
	return fasttime.Int2TS(m[id])
}

// FileChannelID returns the channel ID where the file was last seen.
func (s *State) FileChannelID(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Files[id]
}

func (s *State) SetChunkFilename(filename string) *State {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ChunkFilename = filename
	return s
}

func (s *State) SetFilesDir(dir string) *State {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.FilesDir = dir
	return s
}

func (s *State) SetIsCompressed(isCompressed bool) *State {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.IsCompressed = isCompressed
	return s
}

func (s *State) SetIsComplete(isComplete bool) *State {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.IsComplete = isComplete
	return s
}

// Save saves the state to the given file.
func (s *State) Save(filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(s)
}

// SaveFSA saves the state to the given file in the filesystem adapter.
func (s *State) SaveFSA(fsa fsadapter.FS, filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, err := fsa.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(s)
	if err != nil {
		return err
	}
	return nil
}

// Load loads the state from the given file.
func Load(filename string) (*State, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return load(f)
}

func load(r io.Reader) (*State, error) {
	var s State
	if err := json.NewDecoder(r).Decode(&s); err != nil {
		return nil, err
	}
	if s.Version == 0 {
		// adjust version for incomplete state files
		s.Version = Version
	}
	if Version < s.Version {
		return nil, &ErrStateVersion{Expected: Version, Actual: s.Version}
	}
	return &s, nil
}
