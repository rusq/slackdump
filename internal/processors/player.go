package processors

import (
	"encoding/json"
	"errors"
	"io"
	"sync/atomic"

	"github.com/rusq/slackdump/v2/internal/state"
	"github.com/slack-go/slack"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrExhausted = errors.New("exhausted")
)

// Player replays the events from a file, it is able to emulate the API
// responses, if used in conjunction with the [proctest.Server]. Zero value is
// not usable.st
type Player struct {
	rs io.ReadSeeker

	pointer offsets // current event pointers

	idx        index
	lastOffset atomic.Int64
}

// index holds the index of each event type within the file.  key is the event
// ID, value is the list of offsets for that id in the file.
type index map[string][]int64

// offsets holds the index of the current offset for each event id.
type offsets map[string]int

// NewPlayer creates a new event player from the io.ReadSeeker.
func NewPlayer(rs io.ReadSeeker) (*Player, error) {
	idx, err := indexRecords(rs)
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

// indexRecords indexes the records in the reader and returns an index.
func indexRecords(rs io.Reader) (index, error) {
	idx := make(index)

	dec := json.NewDecoder(rs)

	for i := 0; ; i++ {
		offset := dec.InputOffset() // record current offset

		var event Event
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		idx[event.ID()] = append(idx[event.ID()], offset)
	}
	return idx, nil
}

// Offset returns the last read offset of the record ReadSeeker.
func (p *Player) Offset() int64 {
	return p.lastOffset.Load()
}

// tryGetEvent tries to get the event for the given id.  It returns io.EOF if
// there are no more events for the given id.
func (p *Player) tryGetEvent(id string) (*Event, error) {
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
	var event Event
	if err := json.NewDecoder(p.rs).Decode(&event); err != nil {
		return nil, err
	}
	p.pointer[id]++ // increase the offset pointer for the next call.
	return &event, nil
}

// hasMore returns true if there are more events for the given id.
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
	event, err := p.tryGetEvent(channelID)
	if err != nil {
		return nil, err
	}
	return event.Messages, nil
}

// HasMoreMessages returns true if there are more messages to be read for the
// channel.
func (p *Player) HasMoreMessages(channelID string) bool {
	return p.hasMore(channelID)
}

// Thread returns the messages for the given thread.
func (p *Player) Thread(channelID string, threadTS string) ([]slack.Message, error) {
	id := threadID(channelID, threadTS)
	event, err := p.tryGetEvent(id)
	if err != nil {
		return nil, err
	}
	return event.Messages, nil
}

func (p *Player) HasMoreThreads(channelID string, threadTS string) bool {
	return p.hasMore(threadID(channelID, threadTS))
}

func (p *Player) Reset() error {
	p.pointer = make(offsets)
	_, err := p.rs.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	return nil
}

// Replay replays the events in the reader to the channeler in the order they
// were recorded.  It will reset the state of the Player.
func (p *Player) Replay(c Channeler) error {
	return p.ForEach(func(ev *Event) error {
		if ev == nil {
			return nil
		}
		return p.emit(c, *ev)
	})
}

// ForEach iterates over the events in the reader and calls the function for
// each event.  It will reset the state of the Player.
func (p *Player) ForEach(fn func(ev *Event) error) error {
	if err := p.Reset(); err != nil {
		return err
	}
	defer p.rs.Seek(0, io.SeekStart) // reset offset once we finished.
	dec := json.NewDecoder(p.rs)
	for {
		var evt *Event
		if err := dec.Decode(&evt); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := fn(evt); err != nil {
			return err
		}
	}
	return nil
}

// emit emits the event to the channeler.
func (p *Player) emit(c Channeler, evt Event) error {
	switch evt.Type {
	case EventMessages:
		if err := c.Messages(evt.ChannelID, evt.Messages); err != nil {
			return err
		}
	case EventThreadMessages:
		if err := c.ThreadMessages(evt.ChannelID, *evt.Parent, evt.Messages); err != nil {
			return err
		}
	case EventFiles:
		if err := c.Files(evt.ChannelID, *evt.Parent, evt.IsThreadMessage, evt.Files); err != nil {
			return err
		}
	}
	return nil
}

func (p *Player) State() (*state.State, error) {
	s := state.New()
	p.ForEach(func(ev *Event) error {
		if ev == nil {
			return nil
		}
		if ev.Type == EventFiles {
			for _, f := range ev.Files {
				s.AddFile(ev.ChannelID, f.ID)
			}
		}
		if ev.Type == EventThreadMessages {
			for _, m := range ev.Messages {
				s.AddThread(ev.ChannelID, ev.Parent.ThreadTimestamp, m.Timestamp)
			}
		}
		if ev.Type == EventMessages {
			for _, m := range ev.Messages {
				s.AddMessage(ev.ChannelID, m.Timestamp)
			}
		}
		return nil
	})
	return s, nil
}
