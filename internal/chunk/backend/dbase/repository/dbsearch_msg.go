package repository

import (
	"encoding/json"

	"github.com/rusq/slack"
)

type DBSearchMessage struct {
	ID          int64   `db:"ID"`
	ChunkID     int64   `db:"CHUNK_ID"`
	ChannelID   string  `db:"CHANNEL_ID"`
	ChannelName *string `db:"CHANNEL_NAME,omitempty"`
	TS          string  `db:"TS"`
	Text        *string `db:"TXT,omitempty"`
	IDX         int     `db:"IDX"`
	Data        []byte  `db:"DATA"`
}

func NewDBSearchMessage(chunkID int64, idx int, sm *slack.SearchMessage) (*DBSearchMessage, error) {
	data, err := json.Marshal(sm)
	if err != nil {
		return nil, err
	}
	return &DBSearchMessage{
		ChunkID:     chunkID,
		ChannelID:   sm.Channel.ID,
		ChannelName: orNull(sm.Channel.Name != "", sm.Channel.Name),
		TS:          sm.Timestamp,
		Text:        orNull(sm.Text != "", sm.Text),
		IDX:         idx,
		Data:        data,
	}, nil
}

func (c DBSearchMessage) Val() (slack.SearchMessage, error) {
	return unmarshalt[slack.SearchMessage](c.Data)
}

func (DBSearchMessage) tablename() string {
	return "SEARCH_MESSAGE"
}

func (DBSearchMessage) userkey() []string {
	return slice("CHANNEL_ID")
}

func (DBSearchMessage) columns() []string {
	return []string{"CHUNK_ID", "CHANNEL_ID", "CHANNEL_NAME", "TS", "TXT", "IDX", "DATA"}
}

func (c DBSearchMessage) values() []any {
	return []interface{}{c.ChunkID, c.ChannelID, c.ChannelName, c.TS, c.Text, c.IDX, c.Data}
}

type SearchMessageRepository interface {
	BulkRepository[DBSearchMessage]
}

func NewSearchMessageRepository() SearchMessageRepository {
	return newGenericRepository(DBSearchMessage{})
}
