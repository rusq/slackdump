package repository

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
)

type dbObject interface {
	Table() string
	Columns() []string
	Values() []any
}

type repository[T dbObject] interface {
	Insert(ctx context.Context, conn sqlx.ExtContext, t T) error
	InsertAll(ctx context.Context, pconn PrepareExtContext, files iter.Seq2[T, error]) (int, error)
}

type genericRepository[T dbObject] struct{}

func newGenericRepository[T dbObject]() genericRepository[T] {
	return genericRepository[T]{}
}

// stmtInsert returns the insert statement for entity of type T.  The values are unimportant,
// only column names are used.
func (genericRepository[T]) stmtInsert(t T) string {
	var buf strings.Builder
	buf.WriteString("INSERT INTO ")
	buf.WriteString(t.Table())
	buf.WriteString(" (")
	buf.WriteString(strings.Join(t.Columns(), ","))
	buf.WriteString(") VALUES (")
	buf.WriteString(strings.Join(placeholders(t.Columns()), ","))
	buf.WriteString(")")
	return buf.String()
}

func (r genericRepository[T]) Insert(ctx context.Context, conn sqlx.ExtContext, e T) error {
	_, err := conn.ExecContext(ctx, conn.Rebind(r.stmtInsert(e)), e.Values()...)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

func (r genericRepository[T]) InsertAll(ctx context.Context, pconn PrepareExtContext, ee iter.Seq2[T, error]) (int, error) {
	var t T
	stmt, err := pconn.PrepareContext(ctx, pconn.Rebind(r.stmtInsert(t)))
	if err != nil {
		return 0, fmt.Errorf("insert all: prepare %s: %w", t.Table(), err)
	}
	defer stmt.Close()
	var total int
	for m, err := range ee {
		if err != nil {
			return total, fmt.Errorf("insert all: iterator on %s: %w", m.Table(), err)
		}
		if _, err := stmt.ExecContext(ctx, m.Values()...); err != nil {
			return total, fmt.Errorf("insert all %s: %w", m.Table(), err)
		}
		total++
	}
	return total, nil
}
