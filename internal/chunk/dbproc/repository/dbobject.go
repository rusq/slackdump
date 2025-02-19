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
