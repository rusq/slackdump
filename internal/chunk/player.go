package chunk

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/slack-go/slack"
)

var ErrExhausted = errors.New("exhausted")

// Player replays the chunks from a file, it is able to emulate the API
// responses, if used in conjunction with the [proctest.Server]. Zero value is
// not usable.
type Player struct {
	f          *File
	lastOffset atomic.Int64
	pointer    offsets      // current chunk pointers
	ptrMu      sync.RWMutex // pointer mutex
}

func NewPlayerFromFile(cf *File) *Player {
	return &Player{
		f:       cf,
		pointer: make(offsets),
	}
}

func NewPlayer(rs io.ReadSeeker) (*Player, error) {
	cf, err := FromReader(rs)
	if err != nil {
		return nil, err
	}
	return NewPlayerFromFile(cf), nil
}

// Offset returns the last read offset of the record in ReadSeeker.
func (p *Player) Offset() int64 {
	return p.lastOffset.Load()
}

func (p *Player) State() map[GroupID]int {
	p.ptrMu.RLock()
	defer p.ptrMu.RUnlock()
	return p.pointer
}

func (p *Player) SetState(ptrs map[GroupID]int) {
	p.ptrMu.Lock()
	defer p.ptrMu.Unlock()
	p.pointer = ptrs
}

// next tries to get the next chunk for the given id.  It returns
// io.EOF if there are no more chunks for the given id.
func (p *Player) next(id GroupID) (*Chunk, error) {
	p.ptrMu.Lock()
	defer p.ptrMu.Unlock()
	offsets, ok := p.f.Offsets(id)
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
	chunk, err := p.f.chunkAt(offsets[ptr])
	if err != nil {
		return nil, err
	}
	p.pointer[id]++ // increase the offset pointer for the next call.
	return chunk, nil
}

// Messages returns the next message chunk for the given channel.
func (p *Player) Messages(channelID string) ([]slack.Message, error) {
	chunk, err := p.next(GroupID(channelID))
	if err != nil {
		return nil, err
	}
	return chunk.Messages, nil
}

// Users returns the next users chunk.
func (p *Player) Users() ([]slack.User, error) {
	chunk, err := p.next(userChunkID)
	if err != nil {
		return nil, err
	}
	return chunk.Users, nil
}

// Channels returns the next channels chunk.
func (p *Player) Channels() ([]slack.Channel, error) {
	chunk, err := p.next(channelChunkID)
	if err != nil {
		return nil, err
	}
	return chunk.Channels, nil
}

// HasMoreMessages returns true if there are more messages to be read for the
// channel.
func (p *Player) HasMoreMessages(channelID string) bool {
	return p.hasMore(GroupID(channelID))
}

// hasMore returns true if there are more chunks for the given id.
func (p *Player) hasMore(id GroupID) bool {
	p.ptrMu.RLock()
	defer p.ptrMu.RUnlock()
	offsets, ok := p.f.Offsets(id)
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

func (p *Player) HasMoreThreads(channelID string, threadTS string) bool {
	return p.hasMore(threadID(channelID, threadTS))
}

func (p *Player) HasMoreChannels() bool {
	return p.hasMore(channelChunkID)
}

// HasUsers returns true if there is at least one user chunk in the file.
func (p *Player) HasUsers() bool {
	return p.hasMore(userChunkID)
}

func (p *Player) HasChannels() bool {
	return p.hasMore(channelChunkID)
}

// Thread returns the messages for the given thread.
func (p *Player) Thread(channelID string, threadTS string) ([]slack.Message, error) {
	id := threadID(channelID, threadTS)
	chunk, err := p.next(id)
	if err != nil {
		return nil, err
	}
	return append([]slack.Message{*chunk.Parent}, chunk.Messages...), nil
}

// Reset resets the state of the Player.
func (p *Player) Reset() error {
	p.ptrMu.Lock()
	p.pointer = make(offsets)
	p.ptrMu.Unlock()
	return nil
}

// ChannelInfo returns the channel information for the given channel.  It
// returns an error if the channel is not found within the chunkfile.
func (p *Player) ChannelInfo(id string) (*slack.Channel, error) {
	return p.channelInfo(id, false)
}

func (p *Player) ThreadChannelInfo(id string) (*slack.Channel, error) {
	return p.channelInfo(id, true)
}

func (p *Player) channelInfo(id string, isThread bool) (*slack.Channel, error) {
	chunk, err := p.next(channelInfoID(id, isThread))
	if err != nil {
		return nil, err
	}
	return chunk.Channel, nil
}

func (p *Player) WorkspaceInfo() (*slack.AuthTestResponse, error) {
	return p.f.WorkspaceInfo()
}

func (p *Player) Close() error {
	return p.f.Close()
}
