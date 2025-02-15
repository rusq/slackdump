package repository

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

type dbObject interface {
	tablename() string
	columns() []string
	values() []any
}

type repository[T dbObject] interface {
	// Insert should insert the entity into the database.
	Insert(ctx context.Context, conn sqlx.ExtContext, t *T) error
	// InsertAll should insert all entities from the iterator into the database.
	InsertAll(ctx context.Context, pconn PrepareExtContext, tt iter.Seq2[*T, error]) (int, error)
	// CountType should return the number of entities in the database of a given chunk type.
	CountType(ctx context.Context, conn sqlx.QueryerContext, chunkTypeID chunk.ChunkType) (int64, error)
	// Count should return the number of entities in the database.
	Count(ctx context.Context, conn sqlx.QueryerContext) (int64, error)
}

type genericRepository[T dbObject] struct {
	t T // reference type
}

func newGenericRepository[T dbObject](t T) genericRepository[T] {
	return genericRepository[T]{t: t}
}

// stmtInsert returns the insert statement for entity of type T.  The values are unimportant,
// only column names are used.
func (r genericRepository[T]) stmtInsert() string {
	var buf strings.Builder
	buf.WriteString("INSERT INTO ")
	buf.WriteString(r.t.tablename())
	buf.WriteString(" (")
	buf.WriteString(strings.Join(r.t.columns(), ","))
	buf.WriteString(") VALUES (")
	buf.WriteString(strings.Join(placeholders(r.t.columns()), ","))
	buf.WriteString(")")
	return buf.String()
}

const CTypeAny = chunk.CAny

// stmtLatest returns the statement that selects the latest chunk for each
// entity. it is only suitable for dictionary type entries, such as channels or
// users.
func (r genericRepository[T]) stmtLatest(tid chunk.ChunkType) (stmt string, binds []any) {
	var buf strings.Builder
	buf.WriteString("SELECT C.ID, MAX(CHUNK_ID) AS CHUNK_ID FROM ")
	buf.WriteString(r.t.tablename())
	buf.WriteString(" AS C JOIN CHUNK AS CH ON CH.ID = C.CHUNK_ID WHERE 1=1 ")
	if tid != CTypeAny {
		buf.WriteString("AND CH.TYPE_ID = ? ")
		binds = append(binds, tid)
	}
	buf.WriteString("GROUP BY C.ID")
	return buf.String(), binds
}

func (r genericRepository[T]) Insert(ctx context.Context, conn sqlx.ExtContext, e *T) error {
	_, err := conn.ExecContext(ctx, conn.Rebind(r.stmtInsert()), (*e).values()...)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

func (r genericRepository[T]) InsertAll(ctx context.Context, pconn PrepareExtContext, tt iter.Seq2[*T, error]) (int, error) {
	var t T
	stmt, err := pconn.PrepareContext(ctx, pconn.Rebind(r.stmtInsert()))
	if err != nil {
		return 0, fmt.Errorf("insert all: prepare %s: %w", t.tablename(), err)
	}
	defer stmt.Close()
	var total int
	for m, err := range tt {
		if err != nil {
			return total, fmt.Errorf("insert all: iterator on %s: %w", t.tablename(), err)
		}
		if _, err := stmt.ExecContext(ctx, (*m).values()...); err != nil {
			return total, fmt.Errorf("insert all %s: %w", t.tablename(), err)
		}
		total++
	}
	return total, nil
}

// Count is a generic implementation of the Count method for the repository
// that returns all chunks of T.  Concrete / implementations may choose to
// override it to filter by other type of chunk.
func (r genericRepository[T]) Count(ctx context.Context, conn sqlx.QueryerContext) (int64, error) {
	return r.CountType(ctx, conn, CTypeAny)
}

func (r genericRepository[T]) CountType(ctx context.Context, conn sqlx.QueryerContext, typeID chunk.ChunkType) (int64, error) {
	var n int64
	// TODO: no rebind, not critical, but if the database type changes, this will break
	latest, binds := r.stmtLatest(typeID)
	stmt := `SELECT COUNT (1) FROM (` + latest + `) as latest`
	if err := conn.QueryRowxContext(ctx, stmt, binds...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (r genericRepository[T]) All(ctx context.Context, conn sqlx.QueryerContext) (iter.Seq2[T, error], error) {
	return r.AllOfType(ctx, conn, CTypeAny)
}

func (r genericRepository[T]) AllOfType(ctx context.Context, conn sqlx.QueryerContext, typeID chunk.ChunkType) (iter.Seq2[T, error], error) {
	latest, binds := r.stmtLatest(typeID)
	var buf strings.Builder
	buf.WriteString("WITH LATEST AS (\n")
	buf.WriteString(latest)
	buf.WriteString(")\n")
	buf.WriteString("SELECT * FROM LATEST L JOIN ")
	buf.WriteString(r.t.tablename())
	buf.WriteString(" AS T ON T.ID = L.ID AND T.CHUNK_ID = L.CHUNK_ID WHERE 1=1\n")
	buf.WriteString("ORDER BY T.ID")
	stmt := buf.String()
	rows, err := conn.QueryxContext(ctx, stmt, binds...)
	if err != nil {
		return nil, fmt.Errorf("all: %w", err)
	}
	it := func(yield func(T, error) bool) {
		defer rows.Close()
		for rows.Next() {
			var t T
			if err := rows.StructScan(&t); err != nil {
				yield(t, fmt.Errorf("all: %w", err))
				return
			}
			if !yield(t, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(r.t, fmt.Errorf("all: %w", err))
			return
		}
	}
	return it, nil
}
