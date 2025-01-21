package chunk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"path/filepath"
	"runtime/trace"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk/state"
	"github.com/rusq/slackdump/v3/internal/fasttime"
	"github.com/rusq/slackdump/v3/internal/osext"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrDataMisaligned = errors.New("internal error: index and file data misaligned")
)

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

func (idx index) offsetsWithPrefix(prefix string) []int64 {
	var offsets []int64
	for id, off := range idx {
		if len(id) == 0 {
			log.Panicf("internal error:  invalid id: %q", id)
		}
		if strings.HasPrefix(string(id), prefix) {
			offsets = append(offsets, off...)
		}
	}
	return offsets
}

// FromReader creates a new chunk File from the io.ReadSeeker.
func FromReader(rs io.ReadSeeker) (*File, error) {
	if _, err := rs.Seek(0, io.SeekStart); err != nil { // reset offset
		return nil, err
	}
	rgn := trace.StartRegion(context.Background(), "indexing chunks")
	idx, err := indexChunks(json.NewDecoder(rs))
	rgn.End()
	if err != nil {
		return nil, err
	}
	return &File{
		rs:  rs,
		idx: idx,
	}, nil
}

// fromReaderWithIndex creates a new chunk File from the io.ReadSeeker and
// index.
//
// USE WITH CAUTION: It does not check if the file corresponds to the index.
func fromReaderWithIndex(rs io.ReadSeeker, idx index) (*File, error) {
	if _, err := rs.Seek(0, io.SeekStart); err != nil { // reset offset
		return nil, err
	}
	// TODO: validate index.
	return &File{
		rs:  rs,
		idx: idx,
	}, nil
}

// Close closes the underlying reader if it implements io.Closer.
func (f *File) Close() error {
	if c, ok := f.rs.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

type decoder interface {
	Decode(any) error
	InputOffset() int64
}

// indexChunks indexes the records in the reader and returns an index.
func indexChunks(dec decoder) (index, error) {
	start := time.Now()
	idx := make(index, 200) // buffer for 200 chunks to avoid reallocations.
	var id GroupID
	for i := 0; ; i++ {
		offset := dec.InputOffset() // record current offset

		var chunk Chunk
		if err := dec.Decode(&chunk); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		id = chunk.ID()
		idx[id] = append(idx[id], offset)
	}

	slog.Default().Debug("indexing chunks", "len(idx)", len(idx), "caller", osext.Caller(2), "took", time.Since(start).String(), "took", float64(len(idx))/time.Since(start).Seconds())
	return idx, nil
}

// ensure ensures that the file index was generated.
func (f *File) ensure() {
	if f.idx == nil {
		var err error
		f.idx, err = indexChunks(json.NewDecoder(f.rs))
		if err != nil {
			log.Panicf("internal error: %s: index error: %s", osext.Caller(1), err)
		}
	}
}

// Offsets returns all offsets for the given id.
func (f *File) Offsets(id GroupID) ([]int64, bool) {
	f.ensure()
	ret, ok := f.idx[id]
	return ret, ok && len(ret) > 0
}

// HasUsers returns true if there is at least one user chunk in the file.
func (f *File) HasUsers() bool {
	return f.HasChunks(userChunkID)
}

// HasChannels returns true if there is at least one channel chunk in the
// file.
func (f *File) HasChannels() bool {
	return f.HasChunks(channelChunkID)
}

// HasChunks returns true if there is at least one chunk for the given id.
func (f *File) HasChunks(id GroupID) bool {
	f.ensure()
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
			if errors.Is(err, io.EOF) {
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

// State generates and returns the state of the file.  It does not include
// the path to the downloaded files.
func (f *File) State() (*state.State, error) {
	var name string
	if file, ok := f.rs.(osext.Namer); ok {
		name = filepath.Base(file.Name())
	}
	s := state.New(name)
	if err := f.ForEach(func(ev *Chunk) error {
		if ev == nil {
			return nil
		}
		switch ev.Type {
		case CFiles:
			for _, f := range ev.Files {
				// we are adding the files with the empty path as we
				// have no way of knowing if the file was downloaded or not.
				s.AddFile(ev.ChannelID, f.ID, "")
			}
		case CThreadMessages:
			for _, m := range ev.Messages {
				s.AddThread(ev.ChannelID, ev.Parent.ThreadTimestamp, m.Timestamp)
			}
		case CMessages:
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

// AllMessages returns all the messages for the given channel posted to it (no
// thread).  The messages are in the order as they appear in the file.
func (f *File) AllMessages(channelID string) ([]slack.Message, error) {
	m, err := f.allMessagesForID(GroupID(channelID))
	if err != nil {
		return nil, fmt.Errorf("failed getting messages for %s: %w", channelID, err)
	}
	return m, nil
}

// AllThreadMessages returns all the messages for the given thread.  It does
// not return the parent message in the result, use [File.ThreadParent] for
// that.  The messages are in the order as they appear in the file.
func (f *File) AllThreadMessages(channelID, threadTS string) ([]slack.Message, error) {
	return f.allMessagesForID(threadID(channelID, threadTS))
}

// ThreadParent returns the thread parent message for the given thread.  It
// returns ErrNotFound if the thread is not found.
func (f *File) ThreadParent(channelID, threadTS string) (*slack.Message, error) {
	c, err := f.firstChunkForID(threadID(channelID, threadTS))
	if err != nil {
		return nil, fmt.Errorf("parent message: %s:%s: %w", channelID, threadTS, err)
	}
	return c.Parent, nil
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
func (f *File) AllChannelInfos() ([]slack.Channel, error) {
	f.ensure()
	chans, err := allForOffsets(f, f.idx.offsetsWithPrefix(chanInfoPrefix), func(c *Chunk) []slack.Channel {
		return []slack.Channel{*c.Channel}
	})
	if err != nil {
		return nil, err
	}
	for i := range chans {
		if chans[i].IsArchived {
			slog.Default().Debug("skipping archived channel", "i", i, "id", chans[i].ID)
			continue
		}
		members, err := f.ChannelUsers(chans[i].ID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				// ignoring missing channel users
				slog.Default().Warn("no users", "channel_id", chans[i].ID, "error", err)
				continue
			}
			return nil, err
		}
		chans[i].Members = members
	}
	return chans, nil
}

// AllChannelInfoWithMembers returns all channels with Members populated.
func (f *File) AllChannelInfoWithMembers() ([]slack.Channel, error) {
	c, err := f.AllChannelInfos()
	if err != nil {
		return nil, err
	}
	for i := range c {
		members, err := f.ChannelUsers(c[i].ID)
		if err != nil {
			return nil, err
		}
		c[i].Members = members
	}
	return c, nil
}

// int64s implements sort.Interface for []int64.
type int64s []int64

func (a int64s) Len() int           { return len(a) }
func (a int64s) Less(i, j int) bool { return a[i] < a[j] }
func (a int64s) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// allForOffsets returns all the items for the given offsets.
func allForOffsets[T any](p *File, offsets []int64, fn func(c *Chunk) []T) ([]T, error) {
	// sort offsets to prevent random disk access.
	sort.Sort(int64s(offsets))
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
	info, err := f.channelInfo(channelID, false)
	if err != nil {
		return nil, err
	}
	if !info.IsArchived {
		users, err := f.ChannelUsers(channelID)
		if err != nil {
			return nil, fmt.Errorf("failed getting channel users for %q: %w", channelID, err)
		}
		info.Members = users
	}
	return info, nil
}

func (f *File) ChannelUsers(channelID string) ([]string, error) {
	return allForID(f, channelUsersID(channelID), func(c *Chunk) []string {
		return c.ChannelUsers
	})
}

func (f *File) channelInfo(channelID string, _ bool) (*slack.Channel, error) {
	chunk, err := f.firstChunkForID(channelInfoID(channelID))
	if err != nil {
		return nil, err
	}
	if chunk.Channel.ID != channelID {
		return nil, ErrDataMisaligned
	}
	return chunk.Channel, nil
}

// firstChunkForID returns the first chunk in the file for the given id.
func (f *File) firstChunkForID(id GroupID) (*Chunk, error) {
	ofs, ok := f.Offsets(id)
	if !ok {
		return nil, ErrNotFound
	}
	return f.chunkAt(ofs[0])
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
		return nil, fmt.Errorf("chunk %q: %w", id, ErrNotFound)
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

type Result[T any] struct {
	Err error
	Val T
}

// AllChannelIDs returns all the channels in the chunkfile.
func (p *File) AllChannelIDs() []string {
	ids := make([]string, 0, 1)
	for gid := range p.idx {
		id := string(gid)
		if !strings.Contains(id, ":") && !gid.isInfo() && !gid.isList() && !gid.isSearch() {
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
func (f *File) offsetTimestamps(ctx context.Context) (offts, error) {
	ctx, task := trace.NewTask(ctx, "offsetTimestamps")
	defer task.End()

	ret := make(offts, f.idx.OffsetCount())
	for id, offsets := range f.idx {
		switch id[0] {
		case catInfo, catFile, catList, catSearch: // ignoring files, information and list chunks
			continue
		}
		for _, offset := range offsets {
			rgnCA := trace.StartRegion(ctx, "chunkAt")
			chunk, err := f.chunkAt(offset)
			rgnCA.End()
			if err != nil {
				continue
			}
			rgnTS := trace.StartRegion(ctx, "Timestamps")
			ts, err := chunk.Timestamps()
			rgnTS.End()
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

// Addr is the address of a particular message within a chunk file.
type Addr struct {
	Offset int64 // offset within the chunk file
	Index  int16 // index of the message within the messages slice in the chunk
}

// timeOffsets returns a map of the timestamp to the chunk offset and index of
// the message with this timestamp within the message slice.  It converts the
// string timestamp to an int64 timestamp using structures.TS2int, but the
// original string timestamp returned in the TimeOffset struct.
func timeOffsets(ots offts) map[int64]Addr {
	ret := make(map[int64]Addr, len(ots))
	for offset, info := range ots {
		for i, ts := range info.Timestamps {
			ret[ts] = Addr{
				Offset: offset,
				Index:  int16(i),
			}
		}
	}
	return ret
}

// Sorted iterates over all the messages in the chunkfile in chronological
// order.  If desc is true, the slice will be iterated in reverse order.
func (f *File) Sorted(ctx context.Context, desc bool, fn func(ts time.Time, m *slack.Message) error) error {
	ctx, task := trace.NewTask(ctx, "file.Sorted")
	defer task.End()

	rgnOt := trace.StartRegion(ctx, "offsetTimestamps")
	ots, err := f.offsetTimestamps(ctx)
	rgnOt.End()
	if err != nil {
		return err
	}

	rgnTos := trace.StartRegion(ctx, "timeOffsets")
	tos := timeOffsets(ots)
	rgnTos.End()
	tsList := make([]int64, 0, len(tos))
	for ts := range tos {
		tsList = append(tsList, ts)
	}

	trace.WithRegion(ctx, "sorted.sort", func() {
		if desc {
			sort.Sort(sort.Reverse(int64s(tsList)))
		} else {
			sort.Sort(int64s(tsList))
		}
	})

	var (
		prevOffset int64 // previous chunk offset, used to avoid seeking
		chunk      *Chunk
	)
	for _, ts := range tsList {
		tmOff := tos[ts]
		// read the new chunk from the file only in case that the offset has
		// changed.
		if tmOff.Offset != prevOffset {
			var err error
			chunk, err = f.chunkAt(tmOff.Offset)
			if err != nil {
				return err
			}
			prevOffset = tmOff.Offset
		}

		if err := fn(fasttime.Int2Time(ts).UTC(), &chunk.Messages[tmOff.Index]); err != nil {
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
		return nil, fmt.Errorf("seek error: offset %d: %w", offset, err)
	}
	dec := json.NewDecoder(f.rs)
	var chunk *Chunk
	if err := dec.Decode(&chunk); err != nil {
		return nil, fmt.Errorf("decode error: offset %d: %w", offset, err)
	}
	return chunk, nil
}

// WorkspaceInfo returns the workspace info from the chunkfile.
func (f *File) WorkspaceInfo() (*slack.AuthTestResponse, error) {
	chunk, err := f.firstChunkForID(wspInfoChunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get the workspace info: %w", err)
	}

	return chunk.WorkspaceInfo, nil
}
