package processors

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/slack-go/slack"
)

type Player struct {
	rs io.ReadSeeker

	current state // current event pointers

	idx *index
}

type index struct {
	count counts
	// children may not be written in the same order as they are returned by
	// API, therefore we need to keep track of the offset for each child.  The
	// outer map key is the EventType, and the inner map  key is the parent
	// ID.
	children map[EventType]map[string]int64
	// messages are returned sequentially, so we can keep track of the offset for
	// each channel. The key is the channel ID.
	messages map[string][]int64
}

type state struct {
	MessageIdx map[string]int // current message offset INDEX
	Thread     int            // number of threads returned
	File       int            // number of files returned
	threadReq  map[string]int // current thread request. key is thread_ts
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
		current: state{
			MessageIdx: make(map[string]int),
			threadReq:  make(map[string]int),
		},
	}, nil
}

// indexRecords indexes the records in the reader and returns an index.
func indexRecords(rs io.ReadSeeker) (*index, error) {
	var idx = index{
		children: map[EventType]map[string]int64{
			EventThreadMessages: make(map[string]int64),
			EventFiles:          make(map[string]int64),
		},
		messages: make(map[string][]int64),
	}
	dec := json.NewDecoder(rs)
	for i := 0; ; i++ {
		var event Event
		offset := dec.InputOffset() // get current offset

		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		// threadReq is a map of threadTS to the number of requests for that thread
		// number is concatenated to the threadTS to keep track of the order.
		var threadReq = map[string]int{}
		switch event.Type {
		case EventMessages:
			idx.messages[event.ChannelID] = append(idx.messages[event.ChannelID], offset)
			idx.count.Messages++
		case EventThreadMessages:
			id := event.ID()
			idx.children[EventThreadMessages][id+":"+strconv.Itoa(threadReq[id])] = offset
			// increment the counter for this thread
			threadReq[id]++
			idx.count.Threads++
		case EventFiles:
			// technically we don't need these as they're embedded in the
			// messages.  But we'll index them anyway.
			idx.children[EventFiles][event.ID()] = offset
			idx.count.Files++
		}
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil { // reset offset
		return nil, err
	}
	return &idx, nil
}

func (p *Player) Messages(channelID string) ([]slack.Message, error) {
	idx, ok := p.current.MessageIdx[channelID]
	if !ok {
		p.current.MessageIdx[channelID] = 0
	}
	if idx >= p.idx.count.Messages {
		return nil, io.EOF
	}
	offsets, ok := p.idx.messages[channelID]
	if !ok {
		return nil, ErrNotFound
	}
	_, err := p.rs.Seek(offsets[idx], io.SeekStart)
	if err != nil {
		return nil, err
	}
	var event Event
	if err := json.NewDecoder(p.rs).Decode(&event); err != nil {
		return nil, err
	}
	p.current.MessageIdx[channelID]++
	return event.Messages, nil
}

func (p *Player) HasMoreMessages(channelID string) bool {
	return p.current.MessageIdx[channelID] < p.idx.count.Messages
}

var (
	ErrNotFound  = errors.New("not found")
	ErrExhausted = errors.New("exhausted")
)

func (p *Player) Thread(channelID string, threadTS string) ([]slack.Message, error) {
	// check if there are still threads to return
	if p.current.Thread >= p.idx.count.Threads {
		return nil, io.EOF
	}

	id := threadID(channelID, threadTS)
	// BUG: more than 2 chunks of the same threadTS, currently gets overwritten
	// in the indexing.  Needs another map to keep track of the sequence.
	reqNum := p.current.threadReq[id]

	offset, ok := p.idx.children[EventThreadMessages][id+":"+strconv.Itoa(reqNum)]
	if !ok {
		return nil, ErrNotFound
	}
	_, err := p.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}
	p.current.threadReq[id]++ // increase request count for this thread

	var event Event
	if err := json.NewDecoder(p.rs).Decode(&event); err != nil {
		return nil, err
	}
	p.current.Thread++
	return event.Messages, nil
}

func (p *Player) HasMoreThreads(channelID string, threadTS string) bool {
	// check if there are still threads to return
	if p.current.Thread >= p.idx.count.Threads {
		return false
	}

	id := threadID(channelID, threadTS)
	// check if there are more threads for this threadTS
	reqNum := p.current.threadReq[id]
	_, ok := p.idx.children[EventThreadMessages][id+":"+strconv.Itoa(reqNum)]
	return ok
}
