package chunk

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"sort"
	"sync"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/fasttime"
)

// Catalogue is the generic interface for opening a file with a given version.
type Catalogue interface {
	// OpenVersion should open the file with the given version.
	OpenVersion(FileID, int64) (*File, error)
	// FS should return the file system for the catalogue.
	FS() fs.FS
}

// Group is a group of files with the same FileID and different versions.
type Group struct {
	cat Catalogue

	id   FileID
	ver  []int64
	once sync.Once
}

func (g *Group) ensure() {
	g.once.Do(func() {
		sort.Sort(sort.Reverse(int64s(g.ver)))
	})
}

func (g *Group) ChannelInfo() (*slack.Channel, error) {
	g.ensure()

	civ := &channelInfoVersion{Directory: g.cat}
	cis, err := oneRec(g.cat.FS(), civ, FileID(g.id))
	if err != nil {
		return nil, err
	}
	return &cis, nil
}

// grpOffTs is the index of the file and the message timestamps.
type grpOffTs struct {
	idxFile int   // index of the file
	offts   offts // map of the chunk offset to message timestamps
}

// groupAddr is the address of a message within the group of files.
type grpAddr struct {
	idxFile int  // index of the file
	addr    Addr // address within the file
}

func (g *Group) open() (filegroup, error) {
	g.ensure()

	files := make([]*File, len(g.ver))
	for i, v := range g.ver {
		f, err := g.cat.OpenVersion(g.id, v)
		if err != nil {
			return nil, fmt.Errorf("open version %d: %w", v, err)
		}
		files[i] = f
	}
	return files, nil
}

type filegroup []*File

func (fg filegroup) Close() error {
	var err error
	for _, f := range fg {
		if e := f.Close(); e != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}

func (g *Group) SortedIter(ctx context.Context, chanID string, desc bool) (iter.Seq2[int, GroupResult], error) {
	files, err := g.open()
	if err != nil {
		return nil, err
	}
	defer files.Close()

	it, err := g.sorted(ctx, files, chanID, desc)
	if err != nil {
		return nil, err
	}
	return it, nil
}

func (g *Group) Sorted(ctx context.Context, chanID string, desc bool, fn func(ts time.Time, m *slack.Message) error) error {
	files, err := g.open()
	if err != nil {
		return err
	}
	defer files.Close()

	it, err := g.sorted(ctx, files, chanID, desc)
	if err != nil {
		return err
	}
	for i, r := range it {
		if r.Error != nil {
			return r.Error
		}
		if err := fn(r.TS, r.Message); err != nil {
			return fmt.Errorf("fn at %d: %w", i, err)
		}
	}
	return nil
}

// grpMessageIndex is the index of the messages in the group of files.
type grpMessageIndex struct {
	// map of message timestamp to the file index and the address of the message
	// within that file and chunk.  idxFile->addr.Offset->addr.Index
	addrMsg map[int64]grpAddr
	// list of all message timestamps, sorted asc or desc.
	tsList []int64
}

// Sort interface implementation.
func (idx *grpMessageIndex) Len() int           { return len(idx.tsList) }
func (idx *grpMessageIndex) Less(i, j int) bool { return idx.tsList[i] < idx.tsList[j] }
func (idx *grpMessageIndex) Swap(i, j int) {
	idx.tsList[i], idx.tsList[j] = idx.tsList[j], idx.tsList[i]
}

// messageIndex returns the message index for the group of files.
func (fg filegroup) messageIndex(ctx context.Context, chanID string, desc bool) *grpMessageIndex {
	var (
		addrMsg = make(map[int64]grpAddr)
		tsList  []int64
	)

	for i, f := range fg {
		offset2info, err := f.offsetTimestamps(ctx)
		if err != nil {
			return nil
		}
		for ts, off := range timeOffsets(offset2info, chanID) {
			if _, ok := addrMsg[ts]; !ok {
				addrMsg[ts] = grpAddr{idxFile: i, addr: off}
			}
		}
	}
	// we must build it based on the map, as this will exclude duplicates
	for ts := range addrMsg {
		tsList = append(tsList, ts)
	}

	if desc {
		sort.Sort(sort.Reverse(int64s(tsList)))
	} else {
		sort.Sort(int64s(tsList))
	}
	return &grpMessageIndex{addrMsg, tsList}
}

type GroupResult struct {
	TS      time.Time
	Message *slack.Message
	Error   error
}

func (g *Group) sorted(ctx context.Context, files filegroup, chanID string, desc bool) (iter.Seq2[int, GroupResult], error) {
	gmi := files.messageIndex(ctx, chanID, desc)
	if gmi.Len() == 0 {
		return nil, ErrNoData
	}

	// for each message in the list, load the message and call fn with the
	// message
	return func(yield func(int, GroupResult) bool) {
		var (
			prevOffset = make([]int64, len(files))
			chunk      = make([]*Chunk, len(files))
		)
		for i, ts := range gmi.tsList {
			var (
				addr = gmi.addrMsg[ts] // address of the message as (file,chunk,index)

				n = addr.idxFile // file index
				f = files[n]     // select current file
			)

			if prevOffset[n] != addr.addr.Offset || chunk[n] == nil {
				var err error
				chunk[n], err = f.chunkAt(addr.addr.Offset)
				if err != nil {
					yield(i, GroupResult{Error: fmt.Errorf("chunk at %d: %w", addr.addr.Offset, err)})
					return
				}
			}
			if !yield(i, GroupResult{TS: fasttime.Int2Time(ts), Message: &chunk[n].Messages[addr.addr.Index]}) {
				return
			}
		}
	}, nil
}

func (g *Group) AllMessages(ctx context.Context, chanID string) ([]slack.Message, error) {
	it, err := g.SortedIter(ctx, chanID, false)
	if err != nil {
		return nil, err
	}
	var mm []slack.Message
	for _, r := range it {
		if r.Error != nil {
			return nil, r.Error
		}
		mm = append(mm, *r.Message)
	}
	return mm, nil
}

func (g *Group) AllThreadMessages(chanID string, threadTS string) ([]slack.Message, error) {
	tmv := &threadMessageVersion{Directory: g.cat, ChannelID: chanID, ThreadID: threadTS}
	m, err := latestRec(g.cat.FS(), tmv, FileID(g.id))
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (g *Group) Close() error {
	return nil
}
