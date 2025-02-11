package repository

import (
	"github.com/rusq/slack"
)

type DBChannel struct {
	ID       string  `db:"ID"`
	ChunkID  int64   `db:"CHUNK_ID"`
	LoadDTTM string  `db:"LOAD_DTTM,omitempty"`
	Name     *string `db:"NAME"`
	Index    int     `db:"IDX"`
	Data     []byte  `db:"DATA"`
}

func NewDBChannel(chunkID int64, n int, channel *slack.Channel) (*DBChannel, error) {
	data, err := marshal(channel)
	if err != nil {
		return nil, err
	}
	return &DBChannel{
		ID:      channel.ID,
		ChunkID: chunkID,
		Name:    orNull(channel.Name != "", channel.Name),
		Index:   n,
		Data:    data,
	}, nil
}

func (c *DBChannel) Table() string {
	return "CHANNEL"
}

func (c *DBChannel) Columns() []string {
	return []string{"ID", "CHUNK_ID", "NAME", "IDX", "DATA"}
}

func (c *DBChannel) Values() []interface{} {
	return []interface{}{c.ID, c.ChunkID, c.Name, c.Index, c.Data}
}

type ChannelRepository interface {
	repository[*DBChannel]
}

func NewChannelRepository() ChannelRepository {
	return newGenericRepository[*DBChannel]()
}
