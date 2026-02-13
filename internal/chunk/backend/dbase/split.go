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
	"bytes"
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"

	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/chunk"
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
		ThreadOnly:  orNil(ch.Type == chunk.CThreadMessages, ch.ThreadOnly),
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

	slog.DebugContext(ctx, "inserted chunk", "id", id, "len", n, "channel_id", ch.ChannelID, "type", ch.Type, "final", ch.IsLast, "thread_only", ch.ThreadOnly)

	return id, nil
}

// orNil returns a pointer to the value if cond is true, otherwise nil.
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
	case chunk.CMessages, chunk.CThreadMessages:
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
	if len(mm) == 0 {
		return 0, nil
	}
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
	if len(ff) == 0 {
		return 0, nil
	}
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
	if info == nil {
		return 0, errors.New("insertWorkspaceInfo: info is nil")
	}
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

// allUsersIter returns an iterator that will emit all users.
func allUsersIter(dbchunkID int64, users []slack.User) iter.Seq2[*repository.DBUser, error] {
	return func(yield func(*repository.DBUser, error) bool) {
		for i, u := range users {
			if !yield(repository.NewDBUser(dbchunkID, i, &u)) {
				return
			}
		}
	}
}

// newUserIter returns an iterator that will emit only new or updated users.
//
// TODO: It may be desirable to write a test for this.
func newUserIter(ctx context.Context, ur repository.UserRepository, tx repository.PrepareExtContext, dbchunkID int64, users []slack.User) iter.Seq2[*repository.DBUser, error] {
	return func(yield func(*repository.DBUser, error) bool) {
		for i, u := range users {
			newUser, err := repository.NewDBUser(dbchunkID, i, &u)
			if err != nil {
				if !yield(newUser, err) {
					return
				}
				continue
			}
			existing, err := ur.Get(ctx, tx, u.ID)
			if err == nil {
				// the best we can do is to compare the raw JSON of the
				// existing user with the one that is generated by NewDBUser.
				if bytes.EqualFold(existing.Data, newUser.Data) {
					slog.DebugContext(ctx, "user exists, skipping", "id", existing.ID)
					continue
				}
			}
			if !yield(newUser, nil) {
				return
			}
		}
	}
}

// insertUsers inserts users into the database using the connection tx.
// Depending on the options it will insert all or only new or updated users.
func (p *DBP) insertUsers(ctx context.Context, tx repository.PrepareExtContext, dbchunkID int64, users []slack.User) (int, error) {
	if len(users) == 0 {
		return 0, nil
	}
	ur := repository.NewUserRepository()

	var iterFn iter.Seq2[*repository.DBUser, error]
	if p.opts.onlyNewOrChangedUsers {
		iterFn = newUserIter(ctx, ur, tx, dbchunkID, users)
	} else {
		iterFn = allUsersIter(dbchunkID, users)
	}
	return ur.InsertAll(ctx, tx, iterFn)
}

func (*DBP) insertChannels(ctx context.Context, tx repository.PrepareExtContext, dbchunkID int64, channels []slack.Channel) (int, error) {
	if len(channels) == 0 {
		return 0, nil
	}
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
	if len(users) == 0 {
		return 0, nil
	}
	if channelID == "" {
		return 0, errors.New("insertchannelusers: channelID is empty")
	}
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
	if len(mm) == 0 {
		return 0, nil
	}
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
	if len(ff) == 0 {
		return 0, nil
	}
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
