package dirproc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/processor"
)

type ERC struct {
	cd *chunk.Directory
	lg *slog.Logger
	// lazy init
	once opened
	cv   *Conversations
	w    *Workspace
	u    *Users
	c    *Channels
	s    *Search
}

type opened struct {
	cv sync.Once
	w  sync.Once
	u  sync.Once
	c  sync.Once
	s  sync.Once
}

func (e *ERC) Encode(ctx context.Context, chunk chunk.Chunk) error {
	if err := e.writePayload(ctx, chunk); err != nil {
		return fmt.Errorf("encode: %w", err)
	} else {
		slog.DebugContext(ctx, "inserted chunk", "type", chunk.Type)
	}
	return nil
}

// ensure ensure that the relevant processors are created.
func (e *ERC) ensure(c chunk.Chunk) (err error) {
	switch c.Type {
	case chunk.CMessages, chunk.CThreadMessages, chunk.CFiles, chunk.CChannelInfo, chunk.CChannelUsers:
		e.once.cv.Do(func() {
			e.cv, err = NewConversation(e.cd, &processor.NopFiler{}, &chunk.NopTransformer{})
		})
	case chunk.CWorkspaceInfo:
		e.once.w.Do(func() {
			e.w, err = NewWorkspace(e.cd)
		})
	case chunk.CUsers:
		e.once.u.Do(func() {
			e.u, err = NewUsers(e.cd)
		})
	case chunk.CChannels:
		e.once.c.Do(func() {
			e.c, err = NewChannels(e.cd)
		})
	case chunk.CSearchMessages, chunk.CSearchFiles:
		e.once.s.Do(func() {
			e.s, err = NewSearch(e.cd, &processor.NopFiler{})
		})
	}
	return nil
}

func (e *ERC) writePayload(ctx context.Context, c chunk.Chunk) error {
	if err := e.ensure(c); err != nil {
		return fmt.Errorf("writePayload: %w", err)
	}
	switch c.Type {
	case chunk.CMessages:
		return e.cv.Messages(ctx, c.ChannelID, c.NumThreads, c.IsLast, c.Messages)
	case chunk.CThreadMessages:
		return e.cv.ThreadMessages(ctx, c.ChannelID, *c.Parent, c.ThreadOnly, c.IsLast, c.Messages)
	case chunk.CFiles:
		if c.Parent == nil {
			c.Parent = &slack.Message{}
		}
		return e.cv.Files(ctx, c.Channel, *c.Parent, c.Files)
	case chunk.CWorkspaceInfo:
		return e.w.WorkspaceInfo(ctx, c.WorkspaceInfo)
	case chunk.CUsers:
		return e.u.Users(ctx, c.Users)
	case chunk.CChannels:
		return e.c.Channels(ctx, c.Channels)
	case chunk.CChannelInfo:
		return e.cv.ChannelInfo(ctx, c.Channel, c.ThreadTS)
	case chunk.CChannelUsers:
		return e.cv.ChannelUsers(ctx, c.Channel.ID, c.ThreadTS, c.ChannelUsers)
	case chunk.CSearchMessages:
		return e.s.SearchMessages(ctx, c.SearchQuery, c.SearchMessages)
	case chunk.CSearchFiles:
		return e.s.SearchFiles(ctx, c.SearchQuery, c.SearchFiles)
	default:
		return fmt.Errorf("insertpayload: unknown chunk type %v", c.Type)
	}
}

func (e *ERC) IsFinalised(ctx context.Context, channelID string) (bool, error) {
	return e.cv.t.RefCount(chunk.ToFileID(channelID, "", false)) <= 0, nil
}

func (e *ERC) Close() error {
	var errs error
	for _, closer := range []io.Closer{e.cv, e.w, e.u, e.c, e.s} {
		if closer != nil {
			if err := closer.Close(); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
	return errs
}
