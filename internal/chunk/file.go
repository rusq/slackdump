package chunk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime/trace"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/chunk/state"
	"github.com/rusq/slackdump/v2/internal/structures"
)

var ErrNotFound = errors.New("not found")

// File is the catalog of chunks in a file.
type File struct {
	rs   io.ReadSeeker
	rsMu sync.RWMutex

	idx index // index of chunks in the file
}

// index holds the index of each chunk within the file.  key is the chunk ID,
// value is the list of offsets for that id in the file.
type index map[GroupID][]int64

// OffsetCount returns the total number of offsets in the index.
func (idx index) OffsetCount() int {
	n := 0
	for _, offsets := range idx {
		n += len(offsets)
	}
	return n
}

// offsets holds the index of the current offset in the index for each chunk
// ID.
type offsets map[GroupID]int

// FromReader creates a new chunk File from the io.ReadSeeker.
func FromReader(rs io.ReadSeeker) (*File, error) {
	if _, err := rs.Seek(0, io.SeekStart); err != nil { // reset offset
		return nil, err
	}
	idx, err := indexChunks(json.NewDecoder(rs))
	if err != nil {
		return nil, err
	}
	return &File{
		rs:  rs,
		idx: idx,
	}, nil
}

type decoder interface {
	Decode(interface{}) error
	InputOffset() int64
}

// indexChunks indexes the records in the reader and returns an index.
func indexChunks(dec decoder) (index, error) {
	idx := make(index)

	for i := 0; ; i++ {
		offset := dec.InputOffset() // record current offset

		var chunk Chunk
		if err := dec.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		idx[chunk.ID()] = append(idx[chunk.ID()], offset)
	}
	return idx, nil
}

// Offsets returns all offsets for the given id.
func (f *File) Offsets(id GroupID) ([]int64, bool) {
	if f.idx == nil {
		panic("internal error:  File.Offsets called before File.Open")
	}
	ret, ok := f.idx[id]
	return ret, ok
}

func (f *File) HasUsers() bool {
	return f.HasChunks(userChunkID)
}

func (f *File) HasChannels() bool {
	return f.HasChunks(channelChunkID)
}

// HasChunks returns true if there is at least one chunk for the given id.
func (f *File) HasChunks(id GroupID) bool {
	if f.idx == nil {
		panic("internal error:  File.HasChunks called before File.Open")
	}
	_, ok := f.idx[id]
	return ok
}

// ForEach iterates over the chunks in the reader and calls the function for
// each chunk.  It will lock the file until it finishes.
func (f *File) ForEach(fn func(ev *Chunk) error) error {
	// locking mutex for the entire duration of the function, as we actively
	// reading from the reader, and any unexpected Seek may cause issues.
	f.rsMu.Lock()
	defer f.rsMu.Unlock()
	dec := json.NewDecoder(f.rs)
	for {
		var chunk *Chunk
		if err := dec.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := fn(chunk); err != nil {
			return err
		}
	}
	return nil
}

// namer is an interface that allows us to get the name of the file.
type namer interface {
	// Name should return the name of the file.  *os.File implements this
	// interface.
	Name() string
}

// State generates and returns the state of the file.  It does not include
// the path to the downloaded files.
func (f *File) State() (*state.State, error) {
	var name string
	if file, ok := f.rs.(namer); ok {
		name = filepath.Base(file.Name())
	}
	s := state.New(name)
	if err := f.ForEach(func(ev *Chunk) error {
		if ev == nil {
			return nil
		}
		if ev.Type == CFiles {
			for _, f := range ev.Files {
				// we are adding the files with the empty path as we
				// have no way of knowing if the file was downloaded or not.
				s.AddFile(ev.ChannelID, f.ID, "")
			}
		}
		if ev.Type == CThreadMessages {
			for _, m := range ev.Messages {
				s.AddThread(ev.ChannelID, ev.Parent.ThreadTimestamp, m.Timestamp)
			}
		}
		if ev.Type == CMessages {
			for _, m := range ev.Messages {
				s.AddMessage(ev.ChannelID, m.Timestamp)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// AllMessages returns all the messages for the given channel.
func (f *File) AllMessages(channelID string) ([]slack.Message, error) {
	return f.allMessagesForID(GroupID(channelID))
}

// AllThreadMessages returns all the messages for the given thread.
func (f *File) AllThreadMessages(channelID, threadTS string) ([]slack.Message, error) {
	return f.allMessagesForID(threadID(channelID, threadTS))
}

// AllUsers returns all users in the dump file.
func (p *File) AllUsers() ([]slack.User, error) {
	return allForID(p, userChunkID, func(c *Chunk) []slack.User {
		return c.Users
	})
}

// AllChannels returns all channels collected by listing channels in the dump
// file.
func (p *File) AllChannels() ([]slack.Channel, error) {
	return allForID(p, channelChunkID, func(c *Chunk) []slack.Channel {
		return c.Channels
	})
}

// AllChannelInfos returns all the channel information collected by the channel
// info API.
func (p *File) AllChannelInfos() ([]slack.Channel, error) {
	var offsets []int64
	for id, off := range p.idx {
		if len(id) == 0 {
			return nil, fmt.Errorf("internal error:  invalid id: %q", id)
		}
		if id[0] == 'i' {
			offsets = append(offsets, off...)
		}
	}
	return allForOffsets(p, offsets, func(c *Chunk) []slack.Channel {
		return []slack.Channel{*c.Channel}
	})
}

// allForOffsets returns all the items for the given offsets.
func allForOffsets[T any](p *File, offsets []int64, fn func(c *Chunk) []T) ([]T, error) {
	// sort offsets to prevent random access (only applies if there's only one
	// thread for the file)
	sort.Slice(offsets, func(i, j int) bool {
		return offsets[i] < offsets[j]
	})
	var items []T
	for _, offset := range offsets {
		chunk, err := p.chunkAt(offset)
		if err != nil {
			return nil, err
		}
		items = append(items, fn(chunk)...)
	}
	return items, nil
}

// ChannelInfo returns the information for the given channel.
func (f *File) ChannelInfo(channelID string) (*slack.Channel, error) {
	ofs, ok := f.Offsets(channelInfoID(channelID, false))
	if !ok {
		return nil, ErrNotFound
	}
	chunk, err := f.chunkAt(ofs[0])
	if err != nil {
		return nil, err
	}
	if chunk.Channel.ID != channelID {
		return nil, errors.New("internal error, index and file data misaligned")
	}
	return chunk.Channel, nil
}

// allMessagesForID returns all the messages for the given id.
func (f *File) allMessagesForID(id GroupID) ([]slack.Message, error) {
	return allForID(f, id, func(c *Chunk) []slack.Message {
		return c.Messages
	})
}

// allForID returns all the messages for the given id.
func allForID[T any](p *File, id GroupID, fn func(*Chunk) []T) ([]T, error) {
	var ret []T
	offsets, ok := p.idx[id]
	if !ok {
		return nil, ErrNotFound
	}
	for _, offset := range offsets {
		chunk, err := p.chunkAt(offset)
		if err != nil {
			return nil, err
		}
		ret = append(ret, fn(chunk)...)
	}
	return ret, nil
}

// AllChannelIDs returns all the channels in the chunkfile.
func (p *File) AllChannelIDs() []string {
	var ids = make([]string, 0, 1)
	for gid := range p.idx {
		id := string(gid)
		if !strings.Contains(id, ":") && id[0] != 'i' && id[0] != 'l' {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// offts is a mapping of chunk offset to the message timestamps it contains,
// along with some chunk medatata.
type offts map[int64]offsetInfo

// offsetInfo contains the metadata for a chunk and the list of all timestamps
type offsetInfo struct {
	ID         GroupID
	Type       ChunkType
	Timestamps []int64
}

func (o offts) MessageCount() int {
	var count int
	for _, info := range o {
		if info.Type == CMessages {
			count += len(info.Timestamps)
		}
	}
	return count
}

// offsetTimestamp returns a map of the chunk offset to the message timestamps
// it contains.
func (f *File) offsetTimestamps() (offts, error) {
	var ret = make(offts, f.idx.OffsetCount())
	for id, offsets := range f.idx {
		prefix := id[0]
		switch prefix {
		case 'i', 'f', 'l': // ignoring files, information and list chunks
			continue
		}
		for _, offset := range offsets {
			chunk, err := f.chunkAt(offset)
			if err != nil {
				continue
			}
			ts, err := chunk.Timestamps()
			if err != nil {
				return nil, err
			}
			ret[offset] = offsetInfo{
				ID:         chunk.ID(),
				Type:       chunk.Type,
				Timestamps: ts,
			}
		}
	}
	return ret, nil
}

type TimeOffset struct {
	Offset int64 // offset within the chunk file
	Index  int   // index of the message within the messages slice in the chunk
}

// timeOffsets returns a map of the timestamp to the chunk offset and index of
// the message with this timestamp within the message slice.  It converts the
// string timestamp to an int64 timestamp using structures.TS2int, but the
// original string timestamp returned in the TimeOffset struct.
func timeOffsets(ots offts) map[int64]TimeOffset {
	var ret = make(map[int64]TimeOffset, len(ots))
	for offset, info := range ots {
		for i, ts := range info.Timestamps {
			ret[ts] = TimeOffset{
				Offset: offset,
				Index:  i,
			}
		}
	}
	return ret
}

// Sorted iterates over all the messages in the chunkfile in chronological
// order.
func (f *File) Sorted(ctx context.Context, descending bool, fn func(ts time.Time, m *slack.Message) error) error {
	ctx, task := trace.NewTask(ctx, "file.Sorted")
	defer task.End()

	trace.Log(ctx, "mutex", "lock")

	rgnOt := trace.StartRegion(ctx, "offsetTimestamps")
	ots, err := f.offsetTimestamps()
	rgnOt.End()
	if err != nil {
		return err
	}

	rgnTos := trace.StartRegion(ctx, "timeOffsets")
	tos := timeOffsets(ots)
	rgnTos.End()
	var tsList = make([]int64, 0, len(tos))
	for ts := range tos {
		tsList = append(tsList, ts)
	}
	sf := func(i, j int) bool {
		return tsList[i] < tsList[j]
	}
	if descending {
		sf = func(i, j int) bool {
			return tsList[i] > tsList[j]
		}
	}

	sort.Slice(tsList, sf)

	var (
		prevOffset int64 // previous chunk offset, used to avoid seeking
		chunk      *Chunk
	)
	for _, ts := range tsList {
		tmOff := tos[ts]
		// we don't want to be reading the same chunk over and over again.
		if tmOff.Offset != prevOffset {
			var err error
			chunk, err = f.chunkAt(tmOff.Offset)
			if err != nil {
				return err
			}
			prevOffset = tmOff.Offset
		}
		if err := fn(structures.Int2Time(ts).UTC(), &chunk.Messages[tmOff.Index]); err != nil {
			return err
		}
	}
	return nil
}

// chunkAt returns the chunk at the given offset.
func (f *File) chunkAt(offset int64) (*Chunk, error) {
	f.rsMu.Lock()
	defer f.rsMu.Unlock()
	_, err := f.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(f.rs)
	var chunk *Chunk
	if err := dec.Decode(&chunk); err != nil {
		return nil, err
	}
	return chunk, nil
}
