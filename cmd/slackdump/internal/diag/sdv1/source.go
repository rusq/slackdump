package sdv1

import (
	"context"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/source"
)

type Source struct {
	Messages
	path string
	fst  source.Storage
}

func NewSource(path string) (Source, error) {
	if filepath.Ext(path) != ".json" {
		return Source{}, source.ErrNotSupported
	}
	m, err := load(path)
	if err != nil {
		return Source{}, err
	}
	s := Source{
		Messages: m,
	}

	dir := filepath.Dir(path)
	fst, err := source.NewDumpStorage(os.DirFS(dir))
	if err == nil {
		s.fst = fst
	}

	return s, nil
}

func (m Messages) Name() string {
	return m.ChannelID
}

func (m Messages) Type() source.Flags {
	return source.FDump
}

func (m Messages) Channels(ctx context.Context) ([]slack.Channel, error) {
	return m.allChannels(), nil
}

func (m Messages) Users(ctx context.Context) ([]slack.User, error) {
	return m.SD.Users.Users, nil
}

func (m Messages) AllMessages(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	if m.ChannelID != channelID {
		return nil, source.ErrNotFound
	}
	it := func(yield func(slack.Message, error) bool) {
		for _, msg := range m.Messages {
			msg.Blocks = slack.Blocks{} // v1.0.x has damaged blocks
			if !yield(msg, nil) {
				break
			}
		}
	}
	return it, nil
}

func (m Messages) AllThreadMessages(ctx context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	return nil, source.ErrNotFound
}

func (m Messages) Sorted(ctx context.Context, channelID string, desc bool, cb func(ts time.Time, msg *slack.Message) error) error {
	return source.ErrNotSupported
}

func (m Messages) ChannelInfo(ctx context.Context, channelID string) (*slack.Channel, error) {
	for _, ch := range m.SD.Channels {
		if ch.ID == m.ChannelID {
			return &ch, nil
		}
	}
	// if we don't have channel info, create a fake one
	ci := structures.ChannelFromID(m.ChannelID)
	switch m.ChannelID[0] {
	case 'D':
		ci.IsIM = true
	case 'G':
		ci.IsGroup = true
		ci.Name = m.ChannelID
	case 'C':
		ci.IsChannel = true
	}
	users := make(map[string]struct{})
	for _, m := range m.Messages {
		if m.User != "" {
			if _, ok := users[m.User]; !ok {
				users[m.User] = struct{}{}
				ci.Members = append(ci.Members, m.User)
			}
		}
	}
	sort.Strings(ci.Members)
	return ci, nil
}

func (s Source) Files() source.Storage {
	return s.fst
}

func (s Messages) Avatars() source.Storage {
	return source.NoStorage{}
}

func (m Messages) WorkspaceInfo(ctx context.Context) (*slack.AuthTestResponse, error) {
	return nil, source.ErrNotSupported
}
