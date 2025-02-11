package repository

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/tagops"
)

// Session is a Slackdump archive session entry.
type Session struct {
	ID             int64      `db:"ID,omitempty"`
	CreatedAt      time.Time  `db:"CREATED_AT,omitempty"`
	UpdatedAt      time.Time  `db:"UPDATED_AT,omitempty"`
	ParentID       *int64     `db:"PAR_SESSION_ID,omitempty"`
	FromTS         *time.Time `db:"FROM_TS,omitempty"`
	ToTS           *time.Time `db:"TO_TS,omitempty"`
	Finished       bool       `db:"FINISHED"`
	FilesEnabled   bool       `db:"FILES_ENABLED"`
	AvatarsEnabled bool       `db:"AVATARS_ENABLED"`
	Mode           string     `db:"MODE"`
	Args           string     `db:"ARGS,omitempty"`
}

type SessionRepository interface {
	// Insert should insert a new session into the database. If the session has
	// a parent session, it should verify that the parent session exists. It
	// should return the ID of the newly inserted session.
	Insert(ctx context.Context, conn sqlx.ExtContext, s *Session) (int64, error)
	// Finish should mark a [Session] as finished. It should return the number
	// of rows affected.
	Finish(ctx context.Context, conn sqlx.ExtContext, id int64) (int64, error)
	// Get should retrieve a session from the database by its ID.
	Get(ctx context.Context, conn sqlx.ExtContext, id int64) (*Session, error)
	// Update should update a session in the database. It should return the
	// number of rows affected.
	Update(ctx context.Context, conn sqlx.ExtContext, s *Session) (int64, error)
	// Last should return the last session in the database. If finished is not nil,
	// it should return the last session that is finished or not finished,
	// depending on the value of finished.
	Last(ctx context.Context, conn sqlx.ExtContext, finished *bool) (*Session, error)
}

type sessionRepository struct{}

func NewSessionRepository() SessionRepository {
	return sessionRepository{}
}

func (r sessionRepository) Insert(ctx context.Context, conn sqlx.ExtContext, s *Session) (int64, error) {
	if s.ParentID != nil && *s.ParentID != 0 {
		// verify parent session exists.
		if _, err := r.Get(ctx, conn, *s.ParentID); err != nil {
			return 0, err
		}
	}

	var stmt strings.Builder
	var binds []any
	addbind := newBindAddFn(&stmt, &binds)
	stmt.WriteString("INSERT INTO SESSION (")
	addbind(s.ID > 0, "ID,", s.ID)
	addbind(!s.CreatedAt.IsZero(), "CREATED_AT,", s.CreatedAt)
	addbind(!s.UpdatedAt.IsZero(), "UPDATED_AT,", s.UpdatedAt)
	addbind(s.ParentID != nil && *s.ParentID > 0, "PAR_SESSION_ID,", s.ParentID)
	if s.FromTS != nil && !s.FromTS.IsZero() {
		addbind(true, "FROM_TS,", s.FromTS.UTC())
	}
	if s.ToTS != nil && !s.ToTS.IsZero() {
		addbind(true, "TO_TS,", s.ToTS.UTC())
	}
	stmt.WriteString("FINISHED,FILES_ENABLED,AVATARS_ENABLED,MODE,ARGS) VALUES (")
	binds = append(binds, s.Finished, s.FilesEnabled, s.AvatarsEnabled, s.Mode, s.Args)

	stmt.WriteString(strings.Join(placeholders(binds), ","))
	stmt.WriteString(")")
	slog.Debug("insert", "stmt", stmt.String())

	ret, err := conn.ExecContext(ctx, conn.Rebind(stmt.String()), binds...)
	if err != nil {
		return 0, err
	}
	return ret.LastInsertId()
}

func (r sessionRepository) Finish(ctx context.Context, conn sqlx.ExtContext, id int64) (int64, error) {
	ret, err := conn.ExecContext(ctx, conn.Rebind("UPDATE SESSION SET UPDATED_AT = CURRENT_TIMESTAMP, FINISHED = TRUE WHERE ID = ?"), id)
	if err != nil {
		return 0, err
	}
	affected, err := ret.RowsAffected()
	if err != nil {
		return 0, err
	} else if affected == 0 {
		return 0, sql.ErrNoRows
	}
	return affected, nil
}

var sessCols = tagops.Tags(Session{}, dbTag)

func (r sessionRepository) Get(ctx context.Context, conn sqlx.ExtContext, id int64) (*Session, error) {
	s := new(Session)
	cols := strings.Join(sessCols, ",")
	stmt := "SELECT " + cols + " FROM SESSION WHERE ID = ?"
	slog.Debug("get", "stmt", stmt)
	if err := conn.QueryRowxContext(ctx, conn.Rebind(stmt), id).StructScan(s); err != nil {
		return nil, err
	}
	return s, nil
}

func (r sessionRepository) Update(ctx context.Context, conn sqlx.ExtContext, s *Session) (int64, error) {
	var stmt strings.Builder
	var binds []any
	addbind := newBindAddFn(&stmt, &binds)
	curr, err := r.Get(ctx, conn, s.ID)
	if err != nil {
		return 0, err
	}
	stmt.WriteString("UPDATE SESSION SET UPDATED_AT = CURRENT_TIMESTAMP")
	addbind(s.ParentID != nil && *s.ParentID > 0, ",PAR_SESSION_ID = ?", s.ParentID)
	addbind(s.FromTS != nil && !s.FromTS.IsZero(), ",FROM_TS = ?", s.FromTS)
	addbind(s.ToTS != nil && !s.ToTS.IsZero(), ",TO_TS = ?", s.ToTS)
	addbind(curr.Finished != s.Finished, ",FINISHED = ?,", s.Finished)
	addbind(curr.FilesEnabled != s.FilesEnabled, ",FILES_ENABLED = ?", s.FilesEnabled)
	addbind(curr.AvatarsEnabled != s.AvatarsEnabled, ",AVATARS_ENABLED = ?", s.AvatarsEnabled)
	addbind(s.Mode != "", ",MODE = ?", s.Mode)
	addbind(s.Args != "", ",ARGS = ?", s.Args)
	addbind(true, " WHERE ID = ?", s.ID)

	slog.Debug("update", "stmt", stmt.String())

	ret, err := conn.ExecContext(ctx, conn.Rebind(stmt.String()), binds...)
	if err != nil {
		return 0, err
	}
	return ret.RowsAffected()
}

// Last returns the last session in the database.
func (r sessionRepository) Last(ctx context.Context, conn sqlx.ExtContext, finished *bool) (*Session, error) {
	s := new(Session)
	var stmt strings.Builder
	var binds []any
	stmt.WriteString("SELECT * FROM SESSION")
	if finished != nil {
		stmt.WriteString(" WHERE FINISHED = ?")
		binds = append(binds, *finished)
	}
	stmt.WriteString(" ORDER BY ID DESC LIMIT 1")

	if err := conn.QueryRowxContext(ctx, conn.Rebind(stmt.String()), binds...).StructScan(s); err != nil {
		return nil, err
	}
	return s, nil
}
