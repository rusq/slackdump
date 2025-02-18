package repository

import (
	"github.com/rusq/slack"
)

type DBSearchFile struct {
	ID      int64  `db:"ID,omitempty"`
	ChunkID int64  `db:"CHUNK_ID"`
	FileID  string `db:"FILE_ID"`
	Index   int    `db:"IDX"`
	Data    []byte `db:"DATA"`
}

func NewDBSearchFile(chunkID int64, n int, sf *slack.File) (*DBSearchFile, error) {
	data, err := marshal(sf)
	if err != nil {
		return nil, err
	}
	return &DBSearchFile{
		ChunkID: chunkID,
		FileID:  sf.ID,
		Index:   n,
		Data:    data,
	}, nil
}

func (DBSearchFile) tablename() string {
	return "SEARCH_FILE"
}

func (DBSearchFile) userkey() []string {
	return slice("FILE_ID")
}

func (DBSearchFile) columns() []string {
	return []string{"CHUNK_ID", "FILE_ID", "IDX", "DATA"}
}

func (c DBSearchFile) values() []any {
	return []interface{}{c.ChunkID, c.FileID, c.Index, c.Data}
}

type SearchFileRepository interface {
	BulkRepository[DBSearchFile]
}

func NewSearchFileRepository() SearchFileRepository {
	return newGenericRepository(DBSearchFile{})
}
