package repository

import (
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/fasttime"
)

type DBFile struct {
	ID        string  `db:"ID"`
	ChunkID   int64   `db:"CHUNK_ID"`
	ChannelID string  `db:"CHANNEL_ID"`
	MessageID *int64  `db:"MESSAGE_ID"`
	ThreadID  *int64  `db:"THREAD_ID,omitempty"`
	Index     int     `db:"IDX"`
	Mode      string  `db:"MODE"`
	Filename  *string `db:"FILENAME"`
	URL       *string `db:"URL"`
	Data      []byte  `db:"DATA"`
}

func NewDBFile(chunkID int64, idx int, channelID, threadTS string, parentMsgTS string, file *slack.File) (*DBFile, error) {
	data, err := marshal(file)
	if err != nil {
		return nil, err
	}
	var messageID *int64
	if parentMsgTS != "" {
		ts, err := fasttime.TS2int(parentMsgTS)
		if err != nil {
			return nil, err
		}
		messageID = &ts
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
		MessageID: messageID,
		ThreadID:  threadID,
		Index:     idx,
		Mode:      file.Mode,
		Filename:  orNull(file.Name != "", file.Name),
		URL:       orNull(file.URLPrivateDownload != "", file.URLPrivateDownload),
		Data:      data,
	}, nil
}

func (f DBFile) tablename() string {
	return "FILE"
}

func (f DBFile) userkey() []string {
	return slice("ID")
}

func (f DBFile) columns() []string {
	return []string{"ID", "CHUNK_ID", "CHANNEL_ID", "MESSAGE_ID", "THREAD_ID", "IDX", "MODE", "FILENAME", "URL", "DATA"}
}

func (f DBFile) values() []any {
	return []any{f.ID, f.ChunkID, f.ChannelID, f.MessageID, f.ThreadID, f.Index, f.Mode, f.Filename, f.URL, f.Data}
}

func (f DBFile) Val() (slack.File, error) {
	return unmarshalt[slack.File](f.Data)
}

type FileRepository interface {
	BulkRepository[DBFile]
}

func NewFileRepository() FileRepository {
	return newGenericRepository(DBFile{})
}
