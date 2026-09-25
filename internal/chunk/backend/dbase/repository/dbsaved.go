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
	"context"
	"encoding/json"

	"github.com/rusq/slackdump/v4/internal/edge"
)

type DBSavedItem struct {
	ID        int64   `db:"ID"`
	ChunkID   int64   `db:"CHUNK_ID"`
	ItemID    string  `db:"ITEM_ID"`
	ItemType  string  `db:"ITEM_TYPE"`
	TS        *string `db:"TS,omitempty"`
	State     *string `db:"STATE,omitempty"`
	TodoState *string `db:"TODO_STATE,omitempty"`
	IDX       int     `db:"IDX"`
	Data      []byte  `db:"DATA"`
}

func NewDBSavedItem(chunkID int64, idx int, si *edge.SavedItem) (*DBSavedItem, error) {
	data, err := json.Marshal(si)
	if err != nil {
		return nil, err
	}
	return &DBSavedItem{
		ChunkID:   chunkID,
		ItemID:    si.ItemID,
		ItemType:  si.ItemType,
		TS:        orNull(si.Timestamp != "", si.Timestamp),
		State:     orNull(si.State != "", si.State),
		TodoState: orNull(si.TodoState != "", si.TodoState),
		IDX:       idx,
		Data:      data,
	}, nil
}

func (c DBSavedItem) Val() (edge.SavedItem, error) {
	return unmarshalt[edge.SavedItem](c.Data)
}

func (DBSavedItem) tablename() string {
	return "SAVED_ITEM"
}

func (DBSavedItem) userkey() []string {
	return slice("ITEM_ID")
}

func (DBSavedItem) columns() []string {
	return []string{"CHUNK_ID", "ITEM_ID", "ITEM_TYPE", "TS", "STATE", "TODO_STATE", "IDX", "DATA"}
}

func (c DBSavedItem) values() []any {
	return []any{c.ChunkID, c.ItemID, c.ItemType, c.TS, c.State, c.TodoState, c.IDX, c.Data}
}

//go:generate mockgen -destination=mock_repository/mock_saved_item.go . SavedItemRepository
type SavedItemRepository interface {
	BulkRepository[DBSavedItem]
}

func NewSavedItemRepository() SavedItemRepository {
	return newGenericRepository(DBSavedItem{})
}

// PruneRemovedSavedItems deletes every SAVED_ITEM row whose (ITEM_ID, TS)
// key is not present in currentChunkID, i.e. items no longer in Later at
// all (not just completed or archived, those still appear in the chunk).
func PruneRemovedSavedItems(ctx context.Context, tx PrepareExtContext, currentChunkID int64) (int64, error) {
	res, err := tx.ExecContext(ctx, `
		DELETE FROM SAVED_ITEM
		WHERE (ITEM_ID, COALESCE(TS, '')) NOT IN (
			SELECT ITEM_ID, COALESCE(TS, '') FROM SAVED_ITEM WHERE CHUNK_ID = ?
		)`, currentChunkID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
