package repository

import (
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/fasttime"
)

type DBFile struct {
	ID        string    `db:"ID"`
	ChunkID   int64     `db:"CHUNK_ID"`
	LoadDTTM  time.Time `db:"LOAD_DTTM,omitempty"`
	ChannelID string    `db:"CHANNEL_ID"`
	MessageID int64     `db:"MESSAGE_ID"`
	ThreadID  *int64    `db:"THREAD_ID,omitempty"`
	Index     int       `db:"IDX"`
	Filename  string    `db:"FILENAME"`
	URL       string    `db:"URL"`
	Data      []byte    `db:"DATA"`
}

func NewDBFile(chunkID int64, idx int, channelID, threadTS string, parentMsgTS string, file *slack.File) (*DBFile, error) {
	data, err := marshal(file)
	if err != nil {
		return nil, err
	}
	ts, err := fasttime.TS2int(parentMsgTS)
	if err != nil {
		return nil, err
	}
	var threadID *int64
	if threadTS != "" {
		t, err := fasttime.TS2int(threadTS)
		if err != nil {
			return nil, err
		}
		threadID = &t
	}
	return &DBFile{
		ID:        file.ID,
		ChunkID:   chunkID,
		ChannelID: channelID,
		MessageID: ts,
		ThreadID:  threadID,
		Index:     idx,
		URL:       file.URLPrivate,
		Data:      data,
	}, nil
}

func (f *DBFile) Table() string {
	return "FILE"
}

func (f *DBFile) Columns() []string {
	return []string{"ID", "CHUNK_ID", "CHANNEL_ID", "MESSAGE_ID", "THREAD_ID", "IDX", "FILENAME", "URL", "DATA"}
}

func (f *DBFile) Values() []any {
	return []any{f.ID, f.ChunkID, f.ChannelID, f.MessageID, f.ThreadID, f.Index, f.Filename, f.URL, f.Data}
}

type FileRepository interface {
	repository[*DBFile]
}

func NewFileRepository() FileRepository {
	return newGenericRepository[*DBFile]()
}
