package dbproc

import (
	"context"
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
	// canClose set to false when the connection is passed to the source
	// and should not be closed by the source.
	canClose bool
}

// Connect uses existing connection to the database, it initialises the session
// parameters, and returns an error if it goes not as planned.
func Connect(conn *sqlx.DB) (*Source, error) {
	if err := initDB(context.Background(), conn); err != nil {
		return nil, err
	}
	return &Source{
		conn:     conn,
		canClose: false,
	}, nil
}

// Open attempts to open the database at given path.
func Open(ctx context.Context, path string) (*Source, error) {
	// migrate to the latest
	tmpconn, err := sqlx.Open(repository.Driver, "file:"+path)
	if err != nil {
		return nil, err
	}
	if err := repository.Migrate(ctx, tmpconn.DB, false); err != nil {
		_ = tmpconn.Close()
		return nil, err
	}
	if err := tmpconn.Close(); err != nil {
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

// Close closes the database connection.  It is a noop
// if the [Source] was created with [Connect].
func (s *Source) Close() error {
	if !s.canClose {
		return nil
	}
	return s.conn.Close()
}

func (s *Source) Channels(ctx context.Context) ([]slack.Channel, error) {
	cr := repository.NewChannelRepository()
	it, err := cr.AllOfType(ctx, s.conn, chunk.CChannelInfo)
	if err != nil {
		return nil, err
	}
	return collect(it, preallocSz)
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
