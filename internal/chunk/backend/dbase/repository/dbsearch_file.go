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

func (c DBSearchFile) Val() (slack.File, error) {
	return unmarshalt[slack.File](c.Data)
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

//go:generate mockgen -destination=mock_repository/mock_search_file.go . SearchFileRepository
type SearchFileRepository interface {
	BulkRepository[DBSearchFile]
}

func NewSearchFileRepository() SearchFileRepository {
	return newGenericRepository(DBSearchFile{})
}
