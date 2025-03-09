package dbproc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc/repository"
	"github.com/rusq/slackdump/v3/internal/structures"
)

var (
	ErrInvalidSessionID = errors.New("invalid session ID")
	ErrIncomplete       = errors.New("incomplete session")
)

type Chunker struct {
	SessionID int64
}

func (c *Chunker) ToChunk(ctx context.Context, conn sqlx.ExtContext, e chunk.Encoder) error {
	if c.SessionID < 1 {
		return ErrInvalidSessionID
	}
	sr := repository.NewSessionRepository()
	sess, err := sr.Get(ctx, conn, c.SessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidSessionID
		}
		return err
	}
	if !sess.Finished {
		return ErrIncomplete
	}
	cr := repository.NewChunkRepository()
	it, err := cr.All(ctx, conn, c.SessionID)
	if err != nil {
		return err
	}
	for dbchunk, err := range it {
		if err != nil {
			return err
		}
		fn, ok := assemblers[dbchunk.TypeID]
		if !ok {
			return chunk.ErrUnsupChunkType
		}
		chunk, err := fn(ctx, conn, &dbchunk)
		if err != nil {
			return err
		}
		if err := e.Encode(ctx, *chunk); err != nil {
			return fmt.Errorf("error converting chunk %d[%s]: %w", dbchunk.ID, dbchunk.TypeID, err)
		}
	}
	return nil
}

// assemblers is a map of chunk types to their respective assemblers.
var assemblers = map[chunk.ChunkType]func(context.Context, sqlx.ExtContext, *repository.DBChunk) (*chunk.Chunk, error){
	chunk.CMessages:       asmMessages,
	chunk.CThreadMessages: asmThreadMessages,
	chunk.CFiles:          asmFiles,
	chunk.CUsers:          asmUsers,
	chunk.CChannels:       asmChannels,
	chunk.CChannelInfo:    asmChannelInfo,
	chunk.CWorkspaceInfo:  asmWorkspaceInfo,
	chunk.CChannelUsers:   asmChannelUsers,
	chunk.CSearchMessages: asmSearchMessages,
	chunk.CSearchFiles:    asmSearchFiles,
}

func asmMessages(ctx context.Context, conn sqlx.ExtContext, dbchunk *repository.DBChunk) (*chunk.Chunk, error) {
	mr := repository.NewMessageRepository()
	it, err := mr.AllForChunk(ctx, conn, dbchunk.ID)
	if err != nil {
		return nil, err
	}
	c := dbchunk.Chunk()
	for m, err := range it {
		if err != nil {
			return nil, err
		}
		msg, err := m.Val()
		if err != nil {
			return nil, err
		}
		if structures.IsThreadStart(&msg) {
			c.NumThreads++
		}
		c.Messages = append(c.Messages, msg)
	}
	return c, nil
}

func asmThreadMessages(ctx context.Context, conn sqlx.ExtContext, dbchunk *repository.DBChunk) (*chunk.Chunk, error) {
	mr := repository.NewMessageRepository()
	it, err := mr.AllForChunk(ctx, conn, dbchunk.ID)
	if err != nil {
		return nil, err
	}
	c := dbchunk.Chunk()
	for m, err := range it {
		if err != nil {
			return nil, err
		}
		msg, err := m.Val()
		if err != nil {
			return nil, err
		}
		if c.ThreadTS == "" {
			if m.ThreadTS != nil {
				c.ThreadTS = *m.ThreadTS
			}
		}
		if c.Parent == nil && m.ParentID != nil {
			pm, err := getMessage(ctx, conn, *m.ParentID)
			if err != nil {
				return nil, err
			}
			c.Parent = pm
			continue // skip the parent message, TODO: maybe it should be fixed for the archive format.
		}
		c.Messages = append(c.Messages, msg)
	}
	return c, nil
}

// getMessage returns a single message from the repository.
func getMessage(ctx context.Context, conn sqlx.ExtContext, id int64) (*slack.Message, error) {
	mr := repository.NewMessageRepository()
	pm, err := mr.Get(ctx, conn, id)
	if err != nil {
		return nil, err
	}
	p, err := pm.Val()
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func asmFiles(ctx context.Context, conn sqlx.ExtContext, dbchunk *repository.DBChunk) (*chunk.Chunk, error) {
	fr := repository.NewFileRepository()
	it, err := fr.AllForChunk(ctx, conn, dbchunk.ID)
	if err != nil {
		return nil, err
	}
	c := dbchunk.Chunk()
	for f, err := range it {
		if err != nil {
			return nil, err
		}
		file, err := f.Val()
		if err != nil {
			return nil, err
		}
		if c.Parent == nil && f.MessageID != nil {
			pm, err := getMessage(ctx, conn, *f.MessageID)
			if err != nil {
				return nil, err
			}
			c.Parent = pm
		}
		c.Files = append(c.Files, file)
	}
	return c, nil
}

func asmUsers(ctx context.Context, conn sqlx.ExtContext, dbchunk *repository.DBChunk) (*chunk.Chunk, error) {
	ur := repository.NewUserRepository()
	c := dbchunk.Chunk()
	it, err := ur.AllForChunk(ctx, conn, dbchunk.ID)
	if err != nil {
		return nil, err
	}
	for u, err := range it {
		if err != nil {
			return nil, err
		}
		user, err := u.Val()
		if err != nil {
			return nil, err
		}
		c.Users = append(c.Users, user)
	}
	return c, nil
}

func asmChannels(ctx context.Context, conn sqlx.ExtContext, dbchunk *repository.DBChunk) (*chunk.Chunk, error) {
	cr := repository.NewChannelRepository()
	it, err := cr.AllForChunk(ctx, conn, dbchunk.ID)
	if err != nil {
		return nil, err
	}
	c := dbchunk.Chunk()
	for ch, err := range it {
		if err != nil {
			return nil, err
		}
		channel, err := ch.Val()
		if err != nil {
			return nil, err
		}
		c.Channels = append(c.Channels, channel)
	}
	return c, nil
}

func asmChannelInfo(ctx context.Context, conn sqlx.ExtContext, dbchunk *repository.DBChunk) (*chunk.Chunk, error) {
	cr := repository.NewChannelRepository()
	c := dbchunk.Chunk()
	ch, err := cr.OneForChunk(ctx, conn, dbchunk.ID)
	if err != nil {
		return nil, err
	}
	channel, err := ch.Val()
	if err != nil {
		return nil, err
	}
	c.ChannelID = channel.ID
	c.Channel = &channel
	return c, nil
}

func asmWorkspaceInfo(ctx context.Context, conn sqlx.ExtContext, dbchunk *repository.DBChunk) (*chunk.Chunk, error) {
	wr := repository.NewWorkspaceRepository()
	c := dbchunk.Chunk()
	dw, err := wr.OneForChunk(ctx, conn, dbchunk.ID)
	if err != nil {
		return nil, err
	}
	w, err := dw.Val()
	if err != nil {
		return nil, err
	}
	c.WorkspaceInfo = &w
	return c, nil
}

func asmChannelUsers(ctx context.Context, conn sqlx.ExtContext, dbchunk *repository.DBChunk) (*chunk.Chunk, error) {
	cur := repository.NewChannelUserRepository()
	it, err := cur.AllForChunk(ctx, conn, dbchunk.ID)
	if err != nil {
		return nil, err
	}
	c := dbchunk.Chunk()
	for cu, err := range it {
		if err != nil {
			return nil, err
		}
		c.ChannelUsers = append(c.ChannelUsers, cu.UserID)
	}
	return c, nil
}

func asmSearchMessages(ctx context.Context, conn sqlx.ExtContext, dbchunk *repository.DBChunk) (*chunk.Chunk, error) {
	sr := repository.NewSearchMessageRepository()
	c := dbchunk.Chunk()
	if dbchunk.SearchQuery != nil {
		c.SearchQuery = *dbchunk.SearchQuery
	}
	it, err := sr.AllForChunk(ctx, conn, dbchunk.ID)
	if err != nil {
		return nil, err
	}
	for sm, err := range it {
		if err != nil {
			return nil, err
		}
		sm, err := sm.Val()
		if err != nil {
			return nil, err
		}
		c.SearchMessages = append(c.SearchMessages, sm)
	}
	return c, nil
}

func asmSearchFiles(ctx context.Context, conn sqlx.ExtContext, dbchunk *repository.DBChunk) (*chunk.Chunk, error) {
	sr := repository.NewSearchFileRepository()
	c := dbchunk.Chunk()
	if dbchunk.SearchQuery != nil {
		c.SearchQuery = *dbchunk.SearchQuery
	}
	it, err := sr.AllForChunk(ctx, conn, dbchunk.ID)
	if err != nil {
		return nil, err
	}
	for dsf, err := range it {
		if err != nil {
			return nil, err
		}
		sf, err := dsf.Val()
		if err != nil {
			return nil, err
		}
		c.SearchFiles = append(c.SearchFiles, sf)
	}
	return c, nil
}
