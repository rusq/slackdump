package dbproc

import (
	"context"
	"database/sql"
	"iter"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc/repository"
	"github.com/rusq/slackdump/v3/internal/fasttime"
)

const preallocSz = 100

type Source struct {
	conn *sqlx.DB
}

func Open(path string) (*Source, error) {
	conn, err := sqlx.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	return &Source{conn: conn}, nil
}

func (r *Source) Close() error {
	return r.conn.Close()
}

func OpenDB(conn *sqlx.DB) *Source {
	return &Source{conn: conn}
}

func (s *Source) Channels(ctx context.Context) ([]slack.Channel, error) {
	cr := repository.NewChannelRepository()

	tx, err := s.conn.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	it, err := cr.AllOfType(ctx, s.conn, chunk.CChannelInfo)
	if err != nil {
		return nil, err
	}
	return collect(it, preallocSz)
}

func (s *Source) Users(ctx context.Context) ([]slack.User, error) {
	ur := repository.NewUserRepository()

	tx, err := s.conn.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

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
	tx, err := s.conn.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	mr := repository.NewMessageRepository()
	it, err := mr.AllForID(ctx, s.conn, channelID)
	if err != nil {
		return nil, err
	}
	return valueIter(it), nil
}

func (s *Source) AllThreadMessages(ctx context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	tx, err := s.conn.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	mr := repository.NewMessageRepository()
	it, err := mr.AllForThread(ctx, s.conn, channelID, threadID)
	if err != nil {
		return nil, err
	}
	return valueIter(it), nil
}

func (s *Source) Sorted(ctx context.Context, channelID string, desc bool, cb func(ts time.Time, msg *slack.Message) error) error {
	tx, err := s.conn.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

	tx, err := s.conn.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	c, err := cr.Get(ctx, s.conn, channelID)
	if err != nil {
		return nil, err
	}
	v, err := c.Val()
	return &v, err
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
