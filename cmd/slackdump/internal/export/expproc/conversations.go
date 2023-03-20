package expproc

import (
	"context"
	"sync"

	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
)

// Conversations is a processor that writes the channel and thread messages.
type Conversations struct {
	dir string
	cw  map[string]*channelproc
	mu  sync.RWMutex
}

type channelproc struct {
	*baseproc
	// numThreads is the number of threads are expected to be processed for
	// the given channel.  We keep track of the number of threads, to ensure
	// that we don't close the file until all threads are processed.
	// The channel file can be closed when the number of threads is zero.
	numThreads int
}

func NewConversation(dir string) (*Conversations, error) {
	return &Conversations{dir: dir, cw: make(map[string]*channelproc)}, nil
}

// ensure ensures that the channel file is open and the recorder is
// initialized.
func (p *Conversations) ensure(channelID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.cw[channelID]; ok {
		return nil
	}
	bp, err := newBaseProc(p.dir, channelID)
	if err != nil {
		return err
	}
	p.cw[channelID] = &channelproc{
		baseproc:   bp,
		numThreads: 0,
	}
	return nil
}

// ChannelInfo is called for each channel that is retrieved.
func (p *Conversations) ChannelInfo(ctx context.Context, ci *slack.Channel, isThread bool) error {
	r, err := p.recorder(ci.ID)
	if err != nil {
		return err
	}
	return r.ChannelInfo(ctx, ci, isThread)
}

func (p *Conversations) recorder(channelID string) (*baseproc, error) {
	r, ok := p.cw[channelID]
	if ok {
		return r.baseproc, nil
	}
	if err := p.ensure(channelID); err != nil {
		return nil, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cw[channelID].baseproc, nil
}

// threadCount returns the number of threads that are expected to be
// processed for the given channel.
func (p *Conversations) threadCount(channelID string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if _, ok := p.cw[channelID]; !ok {
		return 0
	}
	return p.cw[channelID].numThreads
}

func (p *Conversations) addThreads(channelID string, n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.cw[channelID]; !ok {
		return
	}
	p.cw[channelID].numThreads += n
}

func (p *Conversations) decThreads(channelID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.cw[channelID]; !ok {
		return
	}
	p.cw[channelID].numThreads--
}

// Messages is called for each message that is retrieved.
func (p *Conversations) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, mm []slack.Message) error {
	r, err := p.recorder(channelID)
	if err != nil {
		return err
	}
	if numThreads > 0 {
		p.addThreads(channelID, numThreads)
	}
	return r.Messages(ctx, channelID, numThreads, isLast, mm)
}

// Files is called for each file that is retrieved. The parent message is
// passed in as well.
func (p *Conversations) Files(ctx context.Context, channelID string, parent slack.Message, isThread bool, ff []slack.File) error {
	r, err := p.recorder(channelID)
	if err != nil {
		return err
	}
	return r.Files(ctx, channelID, parent, isThread, ff)
}

// ThreadMessages is called for each of the thread messages that are
// retrieved. The parent message is passed in as well.
func (p *Conversations) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, isLast bool, tm []slack.Message) error {
	r, err := p.recorder(channelID)
	if err != nil {
		return err
	}
	if err := r.ThreadMessages(ctx, channelID, parent, isLast, tm); err != nil {
		return err
	}
	p.decThreads(channelID)
	return nil
}

// Finalise closes the channel file if there are no more threads to process.
func (p *Conversations) Finalise(channelID string) error {
	if p.threadCount(channelID) > 0 {
		dlog.Printf("channel %s: %d threads left", channelID, p.threadCount(channelID))
		return nil
	}
	dlog.Printf("channel %s: closing channel file", channelID)
	r, err := p.recorder(channelID)
	if err != nil {
		return err
	}
	if err := r.Close(); err != nil {
		return err
	}
	p.mu.Lock()
	delete(p.cw, channelID)
	p.mu.Unlock()
	return nil
}

func (p *Conversations) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, r := range p.cw {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}
