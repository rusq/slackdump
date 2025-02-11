package repository

type DBChannelUser struct {
	ID        string `db:"ID"`
	ChunkID   int64  `db:"CHUNK_ID"`
	LoadDTTM  string `db:"LOAD_DTTM,omitempty"`
	ChannelID string `db:"CHANNEL_ID"`
	Index     int    `db:"IDX"`
}

func NewDBChannelUser(chunkID int64, n int, channelID, userID string) (*DBChannelUser, error) {
	return &DBChannelUser{
		ID:        userID,
		ChunkID:   chunkID,
		ChannelID: channelID,
		Index:     n,
	}, nil
}

func (*DBChannelUser) Table() string {
	return "CHANNEL_USER"
}

func (*DBChannelUser) Columns() []string {
	return []string{"ID", "CHUNK_ID", "CHANNEL_ID", "IDX"}
}

func (c *DBChannelUser) Values() []any {
	return []interface{}{c.ID, c.ChunkID, c.ChannelID, c.Index}
}

type ChannelUserRepository interface {
	repository[*DBChannelUser]
}

func NewChannelUserRepository() ChannelUserRepository {
	return newGenericRepository[*DBChannelUser]()
}
