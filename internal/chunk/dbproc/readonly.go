package dbproc

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slack"

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

	it, err := cr.All(ctx, s.conn)
	if err != nil {
		return nil, err
	}
	ch := make([]slack.Channel, 0, sz)
	for c, err := range it {
		if err != nil {
			return nil, err
		}
		v, err := c.Val()
		if err != nil {
			return nil, err
		}
		ch = append(ch, v)
	}
	return ch, nil
}
