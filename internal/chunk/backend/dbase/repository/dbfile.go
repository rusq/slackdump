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

	"github.com/rusq/slackdump/v4/internal/fasttime"
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

//go:generate mockgen -destination=mock_repository/mock_file.go . FileRepository
type FileRepository interface {
	BulkRepository[DBFile]
}

func NewFileRepository() FileRepository {
	return newGenericRepository(DBFile{})
}
