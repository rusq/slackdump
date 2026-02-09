// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package chunk

import (
	"io"
	"sync"
	"sync/atomic"

	"github.com/rusq/slack"
)

// offsets holds the pointer to the current offset in the File offset index
// for each group ID.
type offsets map[GroupID]int

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
	offsets, ok := p.f.offsets(id)
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
	offsets, ok := p.f.offsets(id)
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
	return chunk.Messages, nil
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
	return p.channelInfo(id)
}

func (p *Player) ChannelUsers(channelID string) ([]string, error) {
	ch, err := p.next(channelUsersID(channelID))
	if err != nil {
		return nil, err
	}
	return ch.ChannelUsers, nil
}

func (p *Player) HasMoreChannelUsers(channelID string) bool {
	return p.hasMore(channelUsersID(channelID))
}

func (p *Player) ThreadChannelInfo(id string) (*slack.Channel, error) {
	return p.channelInfo(id)
}

func (p *Player) channelInfo(id string) (*slack.Channel, error) {
	chunk, err := p.next(channelInfoID(id))
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
