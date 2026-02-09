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
	"testing"

	"github.com/rusq/tagops"
	"github.com/stretchr/testify/assert"
)

func Test_dbObject_definition(t *testing.T) {
	allDbObjects := []dbObject{
		DBChannel{},
		DBChannelUser{},
		DBChunk{},
		DBFile{},
		DBMessage{},
		DBUser{},
		DBSearchFile{},
		DBSearchMessage{},
		DBWorkspace{},
	}
	ops := tagops.New(tagops.Tag("db"))
	for _, obj := range allDbObjects {
		// check number of columns and values
		numCol := len(obj.columns())
		numVal := len(obj.values())
		assert.Equal(t, numCol, numVal, "%T: number of columns (%d) and values (%d) should be equal", obj, numCol, numVal)
		// check if tags match
		tags := ops.Tags(obj)
		for _, col := range obj.columns() {
			assert.Contains(t, tags, col, "%T: column %q should be part of tags", obj, col)
		}
		// check if user key is part of columns
		userKey := obj.userkey()
		for _, key := range userKey {
			assert.Contains(t, obj.columns(), key, "%T: user key %q should be part of columns", obj, key)
		}
	}
}
