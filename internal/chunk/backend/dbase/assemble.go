// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package dbase

import (
	"context"
	"errors"

	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/structures"
)

var (
	// ErrInvalidSessionID is returned when the session ID is invalid.
	ErrInvalidSessionID = errors.New("invalid session ID")
	// ErrIncomplete is returned when the session is incomplete.
	ErrIncomplete = errors.New("incomplete session")
)

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

var (
	rpMsg      = repository.NewMessageRepository()
	rpFile     = repository.NewFileRepository()
	rpUser     = repository.NewUserRepository()
	rpChan     = repository.NewChannelRepository()
	rpWsp      = repository.NewWorkspaceRepository()
	rpChanUser = repository.NewChannelUserRepository()
	rpSrchMsg  = repository.NewSearchMessageRepository()
	rpSrchFile = repository.NewSearchFileRepository()
)

func asmMessages(ctx context.Context, conn sqlx.ExtContext, dbchunk *repository.DBChunk) (*chunk.Chunk, error) {
	it, err := rpMsg.AllForChunk(ctx, conn, dbchunk.ID)
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
	it, err := rpMsg.AllForChunk(ctx, conn, dbchunk.ID)
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
		if c.ThreadTS == "" && m.ThreadTS != nil {
			c.ThreadTS = *m.ThreadTS
		}
		if c.Parent == nil && m.ParentID != nil {
			// not using m[0], because it may not be the first chunk for the thread.
			pm, err := getMessage(ctx, conn, *m.ParentID)
			if err != nil {
				return nil, err
			}
			c.Parent = pm
		}
		c.Messages = append(c.Messages, msg)
	}
	return c, nil
}

// getMessage returns a single message from the repository.
func getMessage(ctx context.Context, conn sqlx.ExtContext, id int64) (*slack.Message, error) {
	pm, err := rpMsg.Get(ctx, conn, id)
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
	it, err := rpFile.AllForChunk(ctx, conn, dbchunk.ID)
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
		// fetch the parent message if it's specified.
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
	c := dbchunk.Chunk()
	it, err := rpUser.AllForChunk(ctx, conn, dbchunk.ID)
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
	it, err := rpChan.AllForChunk(ctx, conn, dbchunk.ID)
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
	c := dbchunk.Chunk()
	ch, err := rpChan.OneForChunk(ctx, conn, dbchunk.ID)
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
	c := dbchunk.Chunk()
	dw, err := rpWsp.OneForChunk(ctx, conn, dbchunk.ID)
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
	it, err := rpChanUser.AllForChunk(ctx, conn, dbchunk.ID)
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
	c := dbchunk.Chunk()
	if dbchunk.SearchQuery != nil {
		c.SearchQuery = *dbchunk.SearchQuery
	}
	it, err := rpSrchMsg.AllForChunk(ctx, conn, dbchunk.ID)
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
	c := dbchunk.Chunk()
	if dbchunk.SearchQuery != nil {
		c.SearchQuery = *dbchunk.SearchQuery
	}
	it, err := rpSrchFile.AllForChunk(ctx, conn, dbchunk.ID)
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
