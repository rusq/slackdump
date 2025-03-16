package dbase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

// InsertChunk inserts a chunk into the database.
func (d *DBP) InsertChunk(ctx context.Context, ch *chunk.Chunk) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	txx, err := d.conn.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("insertchunk: begin: %w", err)
	}
	defer txx.Rollback()

	id, err := d.UnsafeInsertChunk(ctx, txx, ch)
	if err != nil {
		return 0, fmt.Errorf("insertchunk: insert: %w", err)
	}

	if err := txx.Commit(); err != nil {
		return 0, fmt.Errorf("insertchunk: commit: %w", err)
	}

	return id, nil
}

// UnsafeInsertChunk does not lock the DBP and does not commit the transaction.
// It should be used for bulk inserts.  Unsafe for concurrent use.
func (d *DBP) UnsafeInsertChunk(ctx context.Context, txx repository.PrepareExtContext, ch *chunk.Chunk) (int64, error) {
	dc := repository.DBChunk{
		SessionID:   d.sessionID,
		UnixTS:      ch.Timestamp,
		TypeID:      ch.Type,
		NumRecords:  ch.Count,
		ChannelID:   orNil(ch.ChannelID != "", ch.ChannelID),
		SearchQuery: orNil(ch.SearchQuery != "", ch.SearchQuery),
		Final:       ch.IsLast,
	}
	cr := repository.NewChunkRepository()
	id, err := cr.Insert(ctx, txx, &dc)
	if err != nil {
		return 0, fmt.Errorf("insertchunk: insert: %w", err)
	}
	n, err := d.insertPayload(ctx, txx, id, ch)
	if err != nil {
		return 0, fmt.Errorf("insertchunk: payload: %w", err)
	}

	slog.DebugContext(ctx, "inserted chunk", "id", id, "len", n, "channel_id", ch.ChannelID, "type", ch.Type, "final", ch.IsLast)

	return id, nil
}

func orNil[T any](cond bool, v T) *T {
	if cond {
		return &v
	}
	return nil
}

type ErrInvalidPayload struct {
	Type      chunk.ChunkType
	ChannelID string
	Reason    string
}

func (e *ErrInvalidPayload) Error() string {
	return fmt.Sprintf("invalid payload: %v, channel: %s, reason: %s", e.Type, e.ChannelID, e.Reason)
}

// insertPayload calls relevant function to insert the chunk payload.
func (d *DBP) insertPayload(ctx context.Context, tx repository.PrepareExtContext, dbchunkID int64, c *chunk.Chunk) (int, error) {
	switch c.Type {
	case chunk.CMessages:
		return d.insertMessages(ctx, tx, dbchunkID, c.ChannelID, c.Messages)
	case chunk.CThreadMessages:
		// prepend the parent message to the messages slice
		if c.Parent == nil {
			return 0, &ErrInvalidPayload{Type: c.Type, ChannelID: c.ChannelID, Reason: "parent message is nil"}
		}
		c.Messages = append([]slack.Message{*c.Parent}, c.Messages...)
		return d.insertMessages(ctx, tx, dbchunkID, c.ChannelID, c.Messages)
	case chunk.CFiles:
		return d.insertFiles(ctx, tx, dbchunkID, c.ChannelID, c.ThreadTS, c.Parent.Timestamp, c.Files)
	case chunk.CWorkspaceInfo:
		return d.insertWorkspaceInfo(ctx, tx, dbchunkID, c.WorkspaceInfo)
	case chunk.CUsers:
		return d.insertUsers(ctx, tx, dbchunkID, c.Users)
	case chunk.CChannels:
		return d.insertChannels(ctx, tx, dbchunkID, c.Channels)
	case chunk.CChannelInfo:
		if c.Channel == nil {
			return 0, &ErrInvalidPayload{Type: c.Type, ChannelID: c.ChannelID, Reason: "channel is nil"}
		}
		return d.insertChannels(ctx, tx, dbchunkID, []slack.Channel{*c.Channel})
	case chunk.CChannelUsers:
		return d.insertChannelUsers(ctx, tx, dbchunkID, c.ChannelID, c.ChannelUsers)
	case chunk.CSearchMessages:
		return d.insertSearchMessages(ctx, tx, dbchunkID, c.SearchQuery, c.SearchMessages)
	case chunk.CSearchFiles:
		return d.insertSearchFiles(ctx, tx, dbchunkID, c.SearchQuery, c.SearchFiles)
	default:
		return 0, fmt.Errorf("insertpayload: unknown chunk type %v", c.Type)
	}
}

func (*DBP) insertMessages(ctx context.Context, tx repository.PrepareExtContext, dbchunkID int64, channelID string, mm []slack.Message) (int, error) {
	mr := repository.NewMessageRepository()
	iterfn := func(yield func(*repository.DBMessage, error) bool) {
		for i, msg := range mm {
			if !yield(repository.NewDBMessage(dbchunkID, i, channelID, &msg)) {
				return
			}
		}
	}
	return mr.InsertAll(ctx, tx, iterfn)
}

func (*DBP) insertFiles(ctx context.Context, tx repository.PrepareExtContext, dbchunkID int64, channelID, threadTS, parMsgTS string, ff []slack.File) (int, error) {
	fr := repository.NewFileRepository()
	iterfn := func(yield func(*repository.DBFile, error) bool) {
		for i, f := range ff {
			if !yield(repository.NewDBFile(dbchunkID, i, channelID, threadTS, parMsgTS, &f)) {
				return
			}
		}
	}
	return fr.InsertAll(ctx, tx, iterfn)
}

func (p *DBP) insertWorkspaceInfo(ctx context.Context, tx repository.PrepareExtContext, dbchunkID int64, info *slack.AuthTestResponse) (int, error) {
	wr := repository.NewWorkspaceRepository()
	dbw, err := repository.NewDBWorkspace(dbchunkID, info)
	if err != nil {
		return 0, err
	}
	if err := wr.Insert(ctx, tx, dbw); err != nil {
		return 0, err
	}
	return 1, nil
}

func (p *DBP) insertUsers(ctx context.Context, tx repository.PrepareExtContext, dbchunkID int64, users []slack.User) (int, error) {
	ur := repository.NewUserRepository()
	iterfn := func(yield func(*repository.DBUser, error) bool) {
		for i, u := range users {
			if !yield(repository.NewDBUser(dbchunkID, i, &u)) {
				return
			}
		}
	}
	return ur.InsertAll(ctx, tx, iterfn)
}

func (*DBP) insertChannels(ctx context.Context, tx repository.PrepareExtContext, dbchunkID int64, channels []slack.Channel) (int, error) {
	cr := repository.NewChannelRepository()
	iterfn := func(yield func(*repository.DBChannel, error) bool) {
		for i, c := range channels {
			if !yield(repository.NewDBChannel(dbchunkID, i, &c)) {
				return
			}
		}
	}
	return cr.InsertAll(ctx, tx, iterfn)
}

func (*DBP) insertChannelUsers(ctx context.Context, tx repository.PrepareExtContext, dbchunkID int64, channelID string, users []string) (int, error) {
	cur := repository.NewChannelUserRepository()
	iterfn := func(yield func(*repository.DBChannelUser, error) bool) {
		for i, u := range users {
			if !yield(repository.NewDBChannelUser(dbchunkID, i, channelID, u)) {
				return
			}
		}
	}
	return cur.InsertAll(ctx, tx, iterfn)
}

func (*DBP) insertSearchMessages(ctx context.Context, tx repository.PrepareExtContext, dbchunkID int64, _ string, mm []slack.SearchMessage) (int, error) {
	mr := repository.NewSearchMessageRepository()
	iterfn := func(yield func(*repository.DBSearchMessage, error) bool) {
		for i, sm := range mm {
			if !yield(repository.NewDBSearchMessage(dbchunkID, i, &sm)) {
				return
			}
		}
	}
	return mr.InsertAll(ctx, tx, iterfn)
}

func (*DBP) insertSearchFiles(ctx context.Context, tx repository.PrepareExtContext, dbchunkID int64, _ string, ff []slack.File) (int, error) {
	fr := repository.NewSearchFileRepository()
	iterfn := func(yield func(*repository.DBSearchFile, error) bool) {
		for i, sf := range ff {
			if !yield(repository.NewDBSearchFile(dbchunkID, i, &sf)) {
				return
			}
		}
	}
	return fr.InsertAll(ctx, tx, iterfn)
}
