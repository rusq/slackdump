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
