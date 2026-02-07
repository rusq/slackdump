// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package repository

import (
	"encoding/json"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Order is the sort order type.
type Order bool

// Sort order.
const (
	// Asc is an ascending order.
	Asc Order = false
	// Desc is a descending order.
	Desc Order = true

	oAsc  = " ASC"
	oDesc = " DESC"
)

func (o Order) String() string {
	if o {
		return oDesc
	}
	return oAsc
}

const (
	Driver = "sqlite"
	dbTag  = "db"
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

var (
	marshal   = json.Marshal
	unmarshal = json.Unmarshal
)

// unmarshalt is a convenience function to unmarshal data into T.
func unmarshalt[T any](data []byte) (T, error) {
	var t T
	if err := unmarshal(data, &t); err != nil {
		return t, err
	}
	return t, nil
}

// slice is a convenience function to create a slice of T.
func slice[T any](s ...T) []T {
	return s
}

// rebinder is something that can rebind a statement to the database dialect.
type rebinder interface {
	Rebind(string) string
}

// rebind attempts to rebind the statement to the database dialect on a
// supported conn.
func rebind(conn sqlx.QueryerContext, stmt string) string {
	if rb, ok := conn.(rebinder); ok {
		return rb.Rebind(stmt)
	}
	return stmt
}
