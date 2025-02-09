package repository

import (
	"strings"

	"github.com/jmoiron/sqlx"
)

const (
	dbDriver = "sqlite"
	dbTag    = "db"
)

// PrepareExtContext is a combination of sqlx.PreparerContext and sqlx.ExtContext.
type PrepareExtContext interface {
	sqlx.PreparerContext
	sqlx.ExtContext
}

func newBindAddFn(buf *strings.Builder, binds *[]any) func(b bool, expr string, v any) {
	return func(b bool, expr string, v any) {
		if !b {
			return
		}
		buf.WriteString(expr)
		if v != nil {
			*binds = append(*binds, v)
		}
	}
}

func placeholders[T any](v []T) []string {
	s := make([]string, len(v))
	for i := range v {
		s[i] = "?"
	}
	return s
}

// orNull is a convenience function to set optional fields.
func orNull[T any](b bool, t T) *T {
	if b {
		return &t
	}
	return nil
}
