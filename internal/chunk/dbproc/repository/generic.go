package repository

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"runtime/trace"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

type Inserter[T dbObject] interface {
	// Insert should insert the entity into the database.
	Insert(ctx context.Context, conn sqlx.ExtContext, t ...*T) error
	// InsertAll should insert all entities from the iterator into the database.
	InsertAll(ctx context.Context, pconn PrepareExtContext, tt iter.Seq2[*T, error]) (int, error)
}

type Counter[T dbObject] interface {
	// CountType should return the number of entities in the database of a given chunk type.
	CountType(ctx context.Context, conn sqlx.QueryerContext, chunkTypeID chunk.ChunkType) (int64, error)
	// Count should return the number of entities in the database.
	Count(ctx context.Context, conn sqlx.QueryerContext) (int64, error)
}

type Dictionary[T dbObject] interface {
	// AllOfType should return all entities of a given chunk type.
	AllOfType(ctx context.Context, conn sqlx.QueryerContext, chunkTypeID chunk.ChunkType) (iter.Seq2[T, error], error)
	// All should return all entities.
	All(ctx context.Context, conn sqlx.QueryerContext) (iter.Seq2[T, error], error)
}

type Getter[T dbObject] interface {
	// Get should return the entity with the given id.
	Get(ctx context.Context, conn sqlx.ExtContext, id any) (T, error)
	// GetType should return the entity with the given id.
	GetType(ctx context.Context, conn sqlx.ExtContext, ct chunk.ChunkType, id any) (T, error)
}

// BulkRepository is a generic repository interface without the means to select
// individual rows.
type BulkRepository[T dbObject] interface {
	Inserter[T]
	Counter[T]
	Dictionary[T]
	Getter[T]
}

var _ BulkRepository[dbObject] = (*genericRepository[dbObject])(nil)

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
	buf.WriteString(colAlias("", r.t.columns()...))
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
	return r.stmtLatestWhere(tid, queryParams{})
}

func colAlias(alias string, col ...string) string {
	var buf strings.Builder
	var prefix string
	if alias != "" {
		prefix = alias + "."
	}
	buf.WriteString(prefix)
	buf.WriteString(strings.Join(col, ","+prefix))
	return buf.String()
}

func (r genericRepository[T]) stmtLatestWhere(tid chunk.ChunkType, qp queryParams) (string, []any) {
	const alias = "T"
	var buf strings.Builder
	var b []any
	buf.WriteString("SELECT ")
	buf.WriteString(colAlias(alias, r.t.userkey()...))
	buf.WriteString(", MAX(CHUNK_ID) AS CHUNK_ID FROM ")
	buf.WriteString(r.t.tablename())
	buf.WriteString(" AS " + alias + " JOIN CHUNK AS CH ON CH.ID = " + alias + ".CHUNK_ID WHERE 1=1 ")
	if tid != CTypeAny {
		buf.WriteString("AND CH.TYPE_ID = ? ")
		b = append(b, tid)
	}
	if qp.Where != "" {
		buf.WriteString("AND (")
		buf.WriteString(qp.Where)
		buf.WriteString(") ")
		b = append(b, qp.Binds...)
	}
	buf.WriteString("GROUP BY ")
	buf.WriteString(colAlias(alias, r.t.userkey()...))
	return buf.String(), b
}

func (r genericRepository[T]) Get(ctx context.Context, conn sqlx.ExtContext, id any) (T, error) {
	return r.GetType(ctx, conn, CTypeAny, id)
}

func (r genericRepository[T]) GetType(ctx context.Context, conn sqlx.ExtContext, ct chunk.ChunkType, id any) (T, error) {
	latest, binds := r.stmtLatestRows(ct, queryParams{Where: "T.ID = ?", Binds: []any{id}})

	slog.DebugContext(ctx, "get", "stmt", latest, "binds", binds)

	var t T
	if err := conn.QueryRowxContext(ctx, latest, binds...).StructScan(&t); err != nil {
		return t, fmt.Errorf("get: %w", err)
	}
	return t, nil
}

func (r genericRepository[T]) Insert(ctx context.Context, conn sqlx.ExtContext, e ...*T) error {
	ctx, task := trace.NewTask(ctx, "Insert")
	defer task.End()
	trace.Logf(ctx, "parameters", "Insert: %T", r.t)

	stmt := conn.Rebind(r.stmtInsert())
	for _, m := range e {
		_, err := conn.ExecContext(ctx, stmt, (*m).values()...)
		if err != nil {
			return fmt.Errorf("insert: %w", err)
		}
	}
	return nil
}

func (r genericRepository[T]) InsertAll(ctx context.Context, pconn PrepareExtContext, tt iter.Seq2[*T, error]) (int, error) {
	ctx, task := trace.NewTask(ctx, "InsertAll")
	defer task.End()
	trace.Logf(ctx, "parameters", "InsertAll: %T", r.t)

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
	return r.countTypeWhere(ctx, conn, typeID, queryParams{})
}

func (r genericRepository[T]) countTypeWhere(ctx context.Context, conn sqlx.QueryerContext, typeID chunk.ChunkType, qp queryParams) (int64, error) {
	ctx, task := trace.NewTask(ctx, "countTypeWhere")
	defer task.End()
	trace.Logf(ctx, "parameters", "countTypeWhere: %T, typeID=%d, where=%s, binds=%v", r.t, typeID, qp.Where, qp.Binds)

	// TODO: no rebind, not critical, but if the database type changes, this will break
	latest, b := r.stmtLatestWhere(typeID, qp)
	stmt := `SELECT COUNT(1) FROM (` + latest + `) as latest`
	slog.DebugContext(ctx, "count", "stmt", stmt, "binds", b)

	var n int64
	if err := conn.QueryRowxContext(ctx, stmt, b...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (r genericRepository[T]) All(ctx context.Context, conn sqlx.QueryerContext) (iter.Seq2[T, error], error) {
	return r.AllOfType(ctx, conn, CTypeAny)
}

// stmtLatestRows returns the statement that selects the latest chunk for each
// entity.
func (r genericRepository[T]) stmtLatestRows(typeID chunk.ChunkType, qp queryParams) (stmt string, binds []any) {
	latest, binds := r.stmtLatestWhere(typeID, qp)

	var buf strings.Builder
	buf.WriteString("WITH LATEST AS (\n")
	buf.WriteString(latest)
	buf.WriteString(")\n")
	buf.WriteString("SELECT T.")
	buf.WriteString(strings.Join(r.t.columns(), ",T."))
	buf.WriteString(" FROM LATEST L JOIN ")
	buf.WriteString(r.t.tablename())
	buf.WriteString(" AS T ON 1 = 1 ")
	for _, col := range r.t.userkey() {
		buf.WriteString("AND T.")
		buf.WriteString(col)
		buf.WriteString(" = L.")
		buf.WriteString(col)
		buf.WriteString("\n")
	}
	buf.WriteString(" AND T.CHUNK_ID = L.CHUNK_ID JOIN CHUNK CH ON T.CHUNK_ID = CH.ID WHERE 1=1\n")

	return buf.String(), binds
}

// AllOfType returns an iterator that yields all latest rows type T for the
// chunk type typeID.
func (r genericRepository[T]) AllOfType(ctx context.Context, conn sqlx.QueryerContext, typeID chunk.ChunkType) (iter.Seq2[T, error], error) {
	return r.allOfTypeWhere(ctx, conn, typeID, queryParams{})
}

type queryParams struct {
	Where        string
	Binds        []any
	OrderBy      []string
	UserKeyOrder bool
}

// allOfTypeWhere returns an iterator that yields all latest rows type T that
// satisfy the where clause.  If where is empty, all entities are returned.
// Number of binds must match the number of placeholders in the where clause.
// For example, if where is "T.ID = ?" then binds must contain one element.
// Aliases:
// - "C" is the alias for "CHUNK"
// - "T" is the alias for the entity type T table.
func (r genericRepository[T]) allOfTypeWhere(ctx context.Context, conn sqlx.QueryerContext, typeID chunk.ChunkType, qp queryParams) (iter.Seq2[T, error], error) {
	ctx, task := trace.NewTask(ctx, "allOfTypeWhere")
	trace.Logf(ctx, "parameters", "allOfTypeWhere: %T typeID=%d, where=%s, binds=%v", r.t, typeID, qp.Where, qp.Binds)

	latest, binds := r.stmtLatestRows(typeID, qp)

	var buf strings.Builder
	buf.WriteString(latest)
	if qp.Where != "" {
		buf.WriteString(" AND ( ")
		buf.WriteString(qp.Where)
		buf.WriteString(" ) ")
	}
	binds = append(binds, qp.Binds...)
	if qp.UserKeyOrder {
		buf.WriteString(" ORDER BY ")
		buf.WriteString(colAlias("T", r.t.userkey()...))
	} else if len(qp.OrderBy) > 0 {
		buf.WriteString(" ORDER BY ")
		buf.WriteString(strings.Join(qp.OrderBy, ","))
	}

	stmt := buf.String()

	slog.DebugContext(ctx, "allOfTypeWhere", "stmt", stmt, "binds", binds)

	rows, err := conn.QueryxContext(ctx, stmt, binds...)
	if err != nil {
		return nil, fmt.Errorf("all: %w", err)
	}
	it := func(yield func(T, error) bool) {
		defer task.End()
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
