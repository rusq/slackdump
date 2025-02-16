package dbproc

import (
	"context"
	"database/sql"
	"iter"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc/repository"
)

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

	sz, err := cr.Count(ctx, s.conn)
	if err != nil {
		return nil, err
	}

	it, err := cr.AllOfType(ctx, s.conn, chunk.CChannelInfo)
	if err != nil {
		return nil, err
	}
	return collect(it, int(sz))
}

func (s *Source) Users(ctx context.Context) ([]slack.User, error) {
	ur := repository.NewUserRepository()

	tx, err := s.conn.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	sz, err := ur.Count(ctx, s.conn)
	if err != nil {
		return nil, err
	}

	it, err := ur.AllOfType(ctx, s.conn, chunk.CUsers)
	if err != nil {
		return nil, err
	}
	return collect(it, int(sz))
}

type valuer[T any] interface {
	Val() (T, error)
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

func (s *Source) AllMessages(ctx context.Context, channelID string) ([]slack.Message, error) {
	tx, err := s.conn.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	mr := repository.NewMessageRepository()
	sz, err := mr.Count(ctx, s.conn, channelID)
	if err != nil {
		return nil, err
	}

	it, err := mr.AllForID(ctx, s.conn, channelID)
	if err != nil {
		return nil, err
	}
	return collect(it, int(sz))
}

func (s *Source) AllThreadMessages(ctx context.Context, channelID, threadID string) ([]slack.Message, error) {
	tx, err := s.conn.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	mr := repository.NewMessageRepository()
	sz, err := mr.CountThread(ctx, s.conn, channelID, threadID)
	if err != nil {
		return nil, err
	}

	it, err := mr.AllForThread(ctx, s.conn, channelID, threadID)
	if err != nil {
		return nil, err
	}
	return collect(it, int(sz))
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
