package proctest

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

type state struct {
	MessageIdx int            // current message offset INDEX
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
			idx.messages = append(idx.messages, offset)
			idx.count.Messages++
		case EventThreadMessages:
			id := event.Parent.ThreadTimestamp
			idx.children[EventThreadMessages][id+":"+strconv.Itoa(threadReq[id])] = offset
			// increment the counter for this thread
			threadReq[id]++
			idx.count.Threads++
		case EventFiles:
			// technically we don't need these as they're embedded in the
			// messages.  But we'll index them anyway.
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
		return nil, io.EOF
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

func (p *Player) HasMoreMessages() bool {
	return p.current.MessageIdx < p.idx.count.Messages
}

var (
	ErrNotFound  = errors.New("not found")
	ErrExhausted = errors.New("exhausted")
)

func (p *Player) Thread(threadTS string) ([]slack.Message, error) {
	// check if there are still threads to return
	if p.current.Thread >= p.idx.count.Threads {
		return nil, io.EOF
	}

	// BUG: more than 2 chunks of the same threadTS, currently gets overwritten
	// in the indexing.  Needs another map to keep track of the sequence.
	reqNum := p.current.threadReq[threadTS]

	offset, ok := p.idx.children[EventThreadMessages][threadTS+":"+strconv.Itoa(reqNum)]
	if !ok {
		return nil, ErrNotFound
	}
	_, err := p.rs.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}
	p.current.threadReq[threadTS]++ // increase request count for this thread

	var event Event
	if err := json.NewDecoder(p.rs).Decode(&event); err != nil {
		return nil, err
	}
	p.current.Thread++
	return event.Messages, nil
}

func (p *Player) HasMoreThreads(threadTS string) bool {
	// check if there are still threads to return
	if p.current.Thread >= p.idx.count.Threads {
		return false
	}
	// check if there are more threads for this threadTS
	reqNum := p.current.threadReq[threadTS]
	_, ok := p.idx.children[EventThreadMessages][threadTS+":"+strconv.Itoa(reqNum)]
	return ok
}
