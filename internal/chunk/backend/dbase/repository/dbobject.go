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

// dbObject is an interface that should be implemented by all entities that are
// to be stored in the database, if they need to make use of genericRepository.
type dbObject interface {
	// tablename should return the table name.
	tablename() string
	// userkey should return the user key columns.  User key is the key that
	// uniquely identifies the logical entity, and is usually a part of primary
	// key, excluding system column, such as CHUNK_ID.
	userkey() []string
	// columns should return the column names.
	columns() []string
	// values should return the values of the entity.
	values() []any
}

// interface assertions
var (
	_ dbObject = DBChannel{}
	_ dbObject = DBChannelUser{}
	_ dbObject = DBChunk{}
	_ dbObject = DBFile{}
	_ dbObject = DBMessage{}
	_ dbObject = DBUser{}
	_ dbObject = DBSearchFile{}
	_ dbObject = DBSearchMessage{}
	_ dbObject = DBWorkspace{}
)
