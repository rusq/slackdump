package expproc

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/slack-go/slack"
)

type Conversations struct {
	dir string
	cw  map[string]*baseproc
	mu  sync.RWMutex
}

func NewConversation(dir string) (*Conversations, error) {
	return &Conversations{dir: dir}, nil
}

func (p *Conversations) ensure(channelID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.cw[channelID]; ok {
		return nil
	}
	wf, err := os.Create(filepath.Join(p.dir, channelID+".json"))
	if err != nil {
		return err
	}
	r := chunk.NewRecorder(wf)
	p.cw[channelID] = &baseproc{dir: p.dir, wf: wf, Recorder: r}
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
		return r, nil
	}
	if err := p.ensure(channelID); err != nil {
		return nil, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cw[channelID], nil
}

// Messages is called for each message that is retrieved.
func (p *Conversations) Messages(ctx context.Context, channelID string, isLast bool, mm []slack.Message) error {
	r, err := p.recorder(channelID)
	if err != nil {
		return err
	}
	return r.Messages(ctx, channelID, isLast, mm)
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
	return r.ThreadMessages(ctx, channelID, parent, isLast, tm)
}

func (p *Conversations) Finalise(channelID string) error {
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
	for _, r := range p.cw {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}
