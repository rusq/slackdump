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

package auth

import (
	"context"
	"errors"
)

type ctxKey int

const providerKey ctxKey = 0

var ErrNoProvider = errors.New("internal error:  no provider in context")

// FromContext returns the auth provider from the context.
func FromContext(ctx context.Context) (Provider, error) {
	if ctx == nil {
		return nil, errors.New("internal error:  nil context")
	}
	prov, ok := ctx.Value(providerKey).(Provider)
	if !ok {
		return nil, ErrNoProvider
	}
	return prov, nil
}

// WithContext returns context with auth provider.
func WithContext(pctx context.Context, p Provider) context.Context {
	return context.WithValue(pctx, providerKey, p)
}
