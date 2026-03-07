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
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAliasRepository_SetGetAllDelete(t *testing.T) {
	conn := testConn(t)
	r := NewAliasRepository()

	require.NoError(t, r.Set(t.Context(), conn, "C123", "alpha"))

	got, err := r.Get(t.Context(), conn, "C123")
	require.NoError(t, err)
	assert.Equal(t, "C123", got.ChannelID)
	assert.Equal(t, "alpha", got.Alias)
	assert.False(t, got.CreatedAt.IsZero())

	all, err := r.All(t.Context(), conn)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, got.ChannelID, all[0].ChannelID)
	assert.Equal(t, got.Alias, all[0].Alias)

	require.NoError(t, r.Set(t.Context(), conn, "C123", "beta"))

	got, err = r.Get(t.Context(), conn, "C123")
	require.NoError(t, err)
	assert.Equal(t, "beta", got.Alias)

	require.NoError(t, r.Delete(t.Context(), conn, "C123"))

	_, err = r.Get(t.Context(), conn, "C123")
	assert.ErrorIs(t, err, sql.ErrNoRows)

	all, err = r.All(t.Context(), conn)
	require.NoError(t, err)
	assert.Empty(t, all)
}
