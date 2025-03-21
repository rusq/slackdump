package dbase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"runtime/trace"
	"time"

	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fasttime"
	"github.com/rusq/slackdump/v3/internal/structures"
)

const preallocSz = 100 // preallocate slice size

type Source struct {
	conn *sqlx.DB
	// canClose set to false when the connection is passed to the source
	// and should not be closed by the source.
	canClose bool
}

// Open attempts to open the database at given path.
func Open(ctx context.Context, path string) (*Source, error) {
	// migrate to the latest
	if err := migrate(ctx, path); err != nil {
		return nil, err
	}
	conn, err := sqlx.Open(repository.Driver, "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	if err := conn.PingContext(ctx); err != nil {
		return nil, err
	}
	return &Source{conn: conn, canClose: true}, nil
}

func migrate(ctx context.Context, path string) error {
	conn, err := sql.Open(repository.Driver, path)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := repository.Migrate(ctx, conn, false); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "PRAGMA wal_checkpoint"); err != nil {
		return err
	}
	return nil
}

// Close closes the database connection.  It is a noop
// if the [Source] was created with [Connect].
func (s *Source) Close() error {
	if !s.canClose {
		slog.Debug("not closing database connection, it was passed to the source")
		return nil
	}
	slog.Debug("closing database connection")
	if err := s.conn.Close(); err != nil {
		slog.Error("error closing database connection", "error", err)
		return err
	}
	return nil
}

// Channels returns all channels.  If the channel info is not available,
// it will attempt to get all channels.
func (s *Source) Channels(ctx context.Context) ([]slack.Channel, error) {
	cr := repository.NewChannelRepository()
	it, err := cr.AllOfType(ctx, s.conn, chunk.CChannelInfo)
	if err != nil {
		return nil, err
	}
	var chns []slack.Channel
	chns, err = collect(it, preallocSz)
	if err != nil {
		return nil, err
	}
	if len(chns) == 0 {
		// no channel info, try getting all channels
		it, err := cr.AllOfType(ctx, s.conn, chunk.CChannels)
		if err != nil {
			return nil, err
		}
		chns, err = collect(it, preallocSz)
		if err != nil {
			return nil, err
		}
	}
	for _, c := range chns {
		users, err := s.channelUsers(ctx, c.ID, c.NumMembers)
		if err != nil {
			return nil, err
		}
		c.Members = users
	}

	return chns, nil
}

func (s *Source) channelUsers(ctx context.Context, channelID string, prealloc int) ([]string, error) {
	cur := repository.NewChannelUserRepository()
	users, err := cur.GetByChannelID(ctx, s.conn, channelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []string{}, nil
		}
		return nil, err
	}
	us := make([]string, 0, prealloc)
	for c, err := range users {
		if err != nil {
			return nil, err
		}
		us = append(us, c.UserID)
	}
	return us, nil
}

func (s *Source) Users(ctx context.Context) ([]slack.User, error) {
	ur := repository.NewUserRepository()

	it, err := ur.AllOfType(ctx, s.conn, chunk.CUsers)
	if err != nil {
		return nil, err
	}
	return collect(it, preallocSz)
}

type valuer[T any] interface {
	Val() (T, error)
}

func valueIter[T any, D valuer[T]](it iter.Seq2[D, error]) iter.Seq2[T, error] {
	iterFn := func(yield func(T, error) bool) {
		for c, err := range it {
			if err != nil {
				var t T
				yield(t, err)
				return
			}
			if !yield(c.Val()) {
				return
			}
		}
	}
	return iterFn
}

func collect[T any, D valuer[T]](it iter.Seq2[D, error], sz int) ([]T, error) {
	vs := make([]T, 0, sz)
	for c, err := range it {
		if err != nil {
			return nil, err
		}
		v, err := c.Val()
		if err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, nil
}

func (s *Source) AllMessages(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	mr := repository.NewMessageRepository()
	it, err := mr.AllForID(ctx, s.conn, channelID)
	if err != nil {
		return nil, err
	}
	return valueIter(it), nil
}

func (s *Source) AllThreadMessages(ctx context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	mr := repository.NewMessageRepository()
	it, err := mr.AllForThread(ctx, s.conn, channelID, threadID)
	if err != nil {
		return nil, err
	}
	return valueIter(it), nil
}

func (s *Source) Sorted(ctx context.Context, channelID string, desc bool, cb func(ts time.Time, msg *slack.Message) error) error {
	mr := repository.NewMessageRepository()
	it, err := mr.Sorted(ctx, s.conn, channelID, repository.Asc)
	if err != nil {
		return err
	}
	for c, err := range it {
		if err != nil {
			return err
		}
		v, err := c.Val()
		if err != nil {
			return err
		}
		if err := cb(fasttime.Int2Time(c.ID), &v); err != nil {
			return err
		}
	}
	return nil
}

func (s *Source) ChannelInfo(ctx context.Context, channelID string) (*slack.Channel, error) {
	cr := repository.NewChannelRepository()
	c, err := cr.Get(ctx, s.conn, channelID)
	if err != nil {
		return nil, err
	}
	v, err := c.Val()
	if err != nil {
		return nil, err
	}
	users, err := s.channelUsers(ctx, v.ID, v.NumMembers)
	if err != nil {
		return nil, err
	}
	v.Members = users

	return &v, nil
}

func (s *Source) WorkspaceInfo(ctx context.Context) (*slack.AuthTestResponse, error) {
	cr := repository.NewWorkspaceRepository()
	dbw, err := cr.GetWorkspace(ctx, s.conn)
	if err != nil {
		return nil, err
	}
	w, err := dbw.Val()
	return &w, err
}

func (s *Source) Latest(ctx context.Context) (map[structures.SlackLink]time.Time, error) {
	ctx, task := trace.NewTask(ctx, "Latest")
	defer task.End()

	r := repository.NewMessageRepository()
	m := make(map[structures.SlackLink]time.Time, preallocSz)
	slog.DebugContext(ctx, "fetching latest messages")
	itm, err := r.LatestMessages(ctx, s.conn)
	if err != nil {
		return nil, err
	}
	for c, err := range itm {
		if err != nil {
			return nil, err
		}
		sl := structures.SlackLink{
			Channel: c.ChannelID,
		}
		m[sl] = fasttime.Int2Time(c.ID)
	}
	slog.DebugContext(ctx, "fetching latest threads")
	ittm, err := r.LatestThreads(ctx, s.conn)
	if err != nil {
		return nil, err
	}
	for c, err := range ittm {
		if err != nil {
			return nil, err
		}
		sl := structures.SlackLink{
			Channel:  c.ChannelID,
			ThreadTS: c.ThreadTS,
		}
		m[sl] = fasttime.Int2Time(c.ID)
	}
	return m, nil
}

func (src *Source) ToChunk(ctx context.Context, e chunk.Encoder, sessID int64) error {
	if sessID < 1 {
		return ErrInvalidSessionID
	}
	sr := repository.NewSessionRepository()
	sess, err := sr.Get(ctx, src.conn, sessID)
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
	it, err := cr.All(ctx, src.conn, sessID)
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
		chunk, err := fn(ctx, src.conn, &dbchunk)
		if err != nil {
			return err
		}
		if err := e.Encode(ctx, chunk); err != nil {
			return fmt.Errorf("error converting chunk %d[%s]: %w", dbchunk.ID, dbchunk.TypeID, err)
		}
	}
	return nil
}

func (src *Source) Sessions(ctx context.Context) ([]repository.Session, error) {
	sr := repository.NewSessionRepository()
	return sr.All(ctx, src.conn)
}
