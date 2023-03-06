package chunk

import (
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/chunk/state"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrExhausted = errors.New("exhausted")
)

// Player replays the chunks from a file, it is able to emulate the API
// responses, if used in conjunction with the [proctest.Server]. Zero value is
// not usable.st
type Player struct {
	rs io.ReadSeeker

	idx     index   // index of chunks in the file
	pointer offsets // current chunk pointers

	lastOffset atomic.Int64
}

// index holds the index of each chunk within the file.  key is the chunk ID,
// value is the list of offsets for that id in the file.
type index map[string][]int64

// offsets holds the index of the current offset in the index for each chunk
// ID.
type offsets map[string]int

// NewPlayer creates a new chunk player from the io.ReadSeeker.
func NewPlayer(rs io.ReadSeeker) (*Player, error) {
	idx, err := indexRecords(json.NewDecoder(rs))
	if err != nil {
		return nil, err
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil { // reset offset
		return nil, err
	}
	return &Player{
		rs:      rs,
		idx:     idx,
		pointer: make(offsets),
	}, nil
}

type decodeOffsetter interface {
	Decode(interface{}) error
	InputOffset() int64
}

// indexRecords indexes the records in the reader and returns an index.
func indexRecords(dec decodeOffsetter) (index, error) {
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

// Offset returns the last read offset of the record in ReadSeeker.
func (p *Player) Offset() int64 {
	return p.lastOffset.Load()
}

// tryGetChunk tries to get the chunk for the given id.  It returns io.EOF if
// there are no more chunks for the given id.
func (p *Player) tryGetChunk(id string) (*Chunk, error) {
	offsets, ok := p.idx[id]
	if !ok {
		return nil, ErrNotFound
	}
	// getting current offset index for the requested id.
	ptr, ok := p.pointer[id]
	if !ok {
		p.pointer[id] = 0 // initialize, if we see it the first time.
	}
	if ptr >= len(offsets) { // check if we've exhausted the messages
		return nil, io.EOF
	}

	p.lastOffset.Store(offsets[ptr])
	_, err := p.rs.Seek(offsets[ptr], io.SeekStart) // seek to the offset
	if err != nil {
		return nil, err
	}

	var chunk Chunk
	// we have to init new decoder at the current offset, because it's
	// not possible to seek the decoder.
	if err := json.NewDecoder(p.rs).Decode(&chunk); err != nil {
		return nil, err
	}
	p.pointer[id]++ // increase the offset pointer for the next call.
	return &chunk, nil
}

// hasMore returns true if there are more chunks for the given id.
func (p *Player) hasMore(id string) bool {
	offsets, ok := p.idx[id]
	if !ok {
		return false // no such id
	}
	// getting current offset index for the requested id.
	ptr, ok := p.pointer[id]
	if !ok {
		return true // hasn't been accessed yet
	}
	return ptr < len(offsets)
}

// Messages returns the messages for the given channel.
func (p *Player) Messages(channelID string) ([]slack.Message, error) {
	chunk, err := p.tryGetChunk(channelID)
	if err != nil {
		return nil, err
	}
	return chunk.Messages, nil
}

// HasMoreMessages returns true if there are more messages to be read for the
// channel.
func (p *Player) HasMoreMessages(channelID string) bool {
	return p.hasMore(channelID)
}

// Thread returns the messages for the given thread.
func (p *Player) Thread(channelID string, threadTS string) ([]slack.Message, error) {
	id := threadID(channelID, threadTS)
	chunk, err := p.tryGetChunk(id)
	if err != nil {
		return nil, err
	}
	return chunk.Messages, nil
}

func (p *Player) HasMoreThreads(channelID string, threadTS string) bool {
	return p.hasMore(threadID(channelID, threadTS))
}

// Reset resets the state of the Player.
func (p *Player) Reset() error {
	p.pointer = make(offsets)
	_, err := p.rs.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	return nil
}

// ForEach iterates over the chunks in the reader and calls the function for
// each chunk.  It will reset the state of the Player.
func (p *Player) ForEach(fn func(ev *Chunk) error) error {
	if err := p.Reset(); err != nil {
		return err
	}
	defer p.rs.Seek(0, io.SeekStart) // reset offset once we finished.
	dec := json.NewDecoder(p.rs)
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

type namer interface {
	Name() string
}

// State generates and returns the state of the player.  It does not include
// the path to the downloaded files.
func (p *Player) State() (*state.State, error) {
	var name string
	if file, ok := p.rs.(namer); ok {
		name = filepath.Base(file.Name())
	}
	s := state.New(name)
	if err := p.ForEach(func(ev *Chunk) error {
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

// allMessagesForID returns all the messages for the given id. It will reset
// the Player prior to execution.
func (p *Player) allMessagesForID(id string) ([]slack.Message, error) {
	if err := p.Reset(); err != nil {
		return nil, err
	}
	var m []slack.Message
	for {
		chunk, err := p.tryGetChunk(id)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		m = append(m, chunk.Messages...)
	}
	return m, nil
}

// AllMessages returns all the messages for the given channel.
func (p *Player) AllMessages(channelID string) ([]slack.Message, error) {
	return p.allMessagesForID(channelID)
}

// AllThreadMessages returns all the messages for the given thread.
func (p *Player) AllThreadMessages(channelID, threadTS string) ([]slack.Message, error) {
	return p.allMessagesForID(threadID(channelID, threadTS))
}

// AllChannels returns all the channels in the chunkfile.
func (p *Player) AllChannels() []string {
	var ids = make([]string, 0, 1)
	for id := range p.idx {
		if !strings.Contains(id, ":") && !strings.HasPrefix(id, "ci") {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// ChannelInfo returns the channel information for the given channel.  It
// returns an error if the channel is not found within the chunkfile.
func (p *Player) ChannelInfo(id string) (*slack.Channel, error) {
	chunk, err := p.tryGetChunk(channelID(id, false))
	if err != nil {
		return nil, err
	}
	return chunk.Channel, nil
}
