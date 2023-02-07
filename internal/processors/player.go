package processors

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/slack-go/slack"
)

type Player struct {
	rs io.ReadSeeker

	current state // current event pointers

	idx *index
}

type state struct {
	MessageIdx int // current message offset INDEX
	Thread     int // number of threads returned
	File       int // number of files returned
}

// counts holds total event counts for each event type.
type counts struct {
	Messages int
	Threads  int
	Files    int
}

func NewPlayer(rs io.ReadSeeker) (*Player, error) {
	idx, err := indexRecords(rs)
	if err != nil {
		return nil, err
	}
	return &Player{
		rs:  rs,
		idx: idx,
	}, nil
}

type index struct {
	count counts
	// children may not be written in the same order as they are returned by
	// API, therefore we need to keep track of the offset for each child.
	children map[EventType]map[string]int64
	// messages are returned sequentially, so we can keep track of the offset
	messages []int64
}

func indexRecords(rs io.ReadSeeker) (*index, error) {
	var idx = index{
		children: map[EventType]map[string]int64{
			EventThreadMessages: make(map[string]int64),
			EventFiles:          make(map[string]int64),
		},
	}
	dec := json.NewDecoder(rs)
	for i := 0; ; i++ {
		var event Event
		offset, err := rs.Seek(0, io.SeekCurrent) // get current offset
		if err != nil {
			return nil, err
		}
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch event.Type {
		case EventMessages:
			idx.messages = append(idx.messages, offset)
			idx.count.Messages++
		case EventThreadMessages:
			idx.children[EventThreadMessages][event.Parent.ThreadTimestamp] = offset
			idx.count.Threads++
		case EventFiles:
			idx.children[EventFiles][event.Parent.ThreadTimestamp] = offset
			idx.count.Files++
		}
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil { // reset offset
		return nil, err
	}
	return &idx, nil
}

func (p *Player) Messages() ([]slack.Message, error) {
	if p.current.MessageIdx >= p.idx.count.Messages {
		return nil, ErrExhausted
	}
	offset := p.idx.messages[p.current.MessageIdx]
	_, err := p.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}
	var event Event
	if err := json.NewDecoder(p.rs).Decode(&event); err != nil {
		return nil, err
	}
	p.current.MessageIdx++
	return event.Messages, nil
}

var (
	ErrNotFound  = errors.New("not found")
	ErrExhausted = errors.New("exhausted")
)

func (p *Player) Thread(threadTS string) ([]slack.Message, error) {
	// check if there are still threads to return
	if p.current.Thread >= p.idx.count.Threads {
		return nil, ErrExhausted
	}

	// BUG: more than 2 chunks of the same threadTS, currently gets overwritten
	// in the indexing.  Needs another map to keep track of the sequence.
	offset, ok := p.idx.children[EventThreadMessages][threadTS]
	if !ok {
		return nil, ErrNotFound
	}
	_, err := p.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}
	var event Event
	if err := json.NewDecoder(p.rs).Decode(&event); err != nil {
		return nil, err
	}
	p.current.Thread++
	return event.Messages, nil
}
