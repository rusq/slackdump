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

package chunk

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"testing"

	"github.com/rusq/slack"
)

func TestChunk_ID(t *testing.T) {
	type fields struct {
		Type      ChunkType
		Timestamp int64
		ThreadTS  string
		Count     int32
		Channel   *slack.Channel
		ChannelID string
		Parent    *slack.Message
		Messages  []slack.Message
		Files     []slack.File
		Users     []slack.User
	}
	tests := []struct {
		name   string
		fields fields
		want   GroupID
	}{
		{
			name: "messages",
			fields: fields{
				Type:      CMessages,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				ChannelID: "C123",
			},
			want: "C123",
		},
		{
			name: "threads",
			fields: fields{
				Type:      CThreadMessages,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				ChannelID: "C123",
				Parent: &slack.Message{
					Msg: slack.Msg{ThreadTimestamp: "1234"},
				},
			},
			want: "tC123:1234",
		},
		{
			name: "files",
			fields: fields{
				Type:      CFiles,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				Parent: &slack.Message{
					Msg: slack.Msg{Timestamp: "1234"},
				},
				ChannelID: "C123",
			},
			want: "fC123:1234",
		},
		{
			name: "channel info",
			fields: fields{
				Type:      CChannelInfo,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				ChannelID: "C123",
			},
			want: "icC123",
		},
		{
			name: "users",
			fields: fields{
				Type:      CUsers,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				ChannelID: "",
			},
			want: userChunkID,
		},
		{
			name: "channels",
			fields: fields{
				Type:      CChannels,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
			},
			want: channelChunkID,
		},
		{
			name:   "workspace info",
			fields: fields{Type: CWorkspaceInfo},
			want:   wspInfoChunkID,
		},
		{
			name:   "starred items",
			fields: fields{Type: CStarredItems},
			want:   starredChunkID,
		},
		{
			name: "bookmarks",
			fields: fields{
				Type:      CBookmarks,
				ChannelID: "C123",
			},
			want: id(bookmarkPrefix, "C123"),
		},
		{
			name:   "search messages",
			fields: fields{Type: CSearchMessages},
			want:   srchMsgChunkID,
		},
		{
			name:   "search files",
			fields: fields{Type: CSearchFiles},
			want:   srchFileChunkID,
		},
		{
			name: "unknown",
			fields: fields{
				Type:      ChunkType(254),
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				ChannelID: "",
			},
			want: "<unknown:ChunkType(254)>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chunk{
				Type:      tt.fields.Type,
				Timestamp: tt.fields.Timestamp,
				ThreadTS:  tt.fields.ThreadTS,
				Count:     tt.fields.Count,
				Channel:   tt.fields.Channel,
				ChannelID: tt.fields.ChannelID,
				Parent:    tt.fields.Parent,
				Messages:  tt.fields.Messages,
				Files:     tt.fields.Files,
				Users:     tt.fields.Users,
			}
			if got := c.ID(); got != tt.want {
				t.Errorf("Chunk.ID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// marshalChunks turns chunks into io.ReadSeeker
func marshalChunks(chunks ...Chunk) io.ReadSeeker {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	for _, c := range chunks {
		if err := enc.Encode(c); err != nil {
			panic(err)
		}
	}
	return bytes.NewReader(b.Bytes())
}

func TestChunk_messageTimestamps(t *testing.T) {
	type fields struct {
		Type           ChunkType
		Timestamp      int64
		ChannelID      string
		Count          int32
		ThreadTS       string
		IsLast         bool
		NumThreads     int32
		Channel        *slack.Channel
		ChannelUsers   []string
		Parent         *slack.Message
		Messages       []slack.Message
		Files          []slack.File
		Users          []slack.User
		Channels       []slack.Channel
		WorkspaceInfo  *slack.AuthTestResponse
		StarredItems   []slack.StarredItem
		Bookmarks      []slack.Bookmark
		SearchQuery    string
		SearchMessages []slack.SearchMessage
		SearchFiles    []slack.File
	}
	tests := []struct {
		name    string
		fields  fields
		want    []int64
		wantErr bool
	}{
		{
			name: "no messages",
			fields: fields{
				Messages: nil,
			},
			want:    []int64{},
			wantErr: false,
		},
		{
			name: "one message",
			fields: fields{
				Messages: []slack.Message{
					{Msg: slack.Msg{Timestamp: "1234.567"}},
				},
			},
			want:    []int64{1234567},
			wantErr: false,
		},
		{
			name: "two messages",
			fields: fields{
				Messages: []slack.Message{
					{Msg: slack.Msg{Timestamp: "1234.567"}},
					{Msg: slack.Msg{Timestamp: "1234.568"}},
				},
			},
			want:    []int64{1234567, 1234568},
			wantErr: false,
		},
		{
			name: "invalid timestamp",
			fields: fields{
				Messages: []slack.Message{
					{Msg: slack.Msg{Timestamp: "1234567"}},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chunk{
				Type:           tt.fields.Type,
				Timestamp:      tt.fields.Timestamp,
				ChannelID:      tt.fields.ChannelID,
				Count:          tt.fields.Count,
				ThreadTS:       tt.fields.ThreadTS,
				IsLast:         tt.fields.IsLast,
				NumThreads:     tt.fields.NumThreads,
				Channel:        tt.fields.Channel,
				ChannelUsers:   tt.fields.ChannelUsers,
				Parent:         tt.fields.Parent,
				Messages:       tt.fields.Messages,
				Files:          tt.fields.Files,
				Users:          tt.fields.Users,
				Channels:       tt.fields.Channels,
				WorkspaceInfo:  tt.fields.WorkspaceInfo,
				StarredItems:   tt.fields.StarredItems,
				Bookmarks:      tt.fields.Bookmarks,
				SearchQuery:    tt.fields.SearchQuery,
				SearchMessages: tt.fields.SearchMessages,
				SearchFiles:    tt.fields.SearchFiles,
			}
			got, err := c.messageTimestamps()
			if (err != nil) != tt.wantErr {
				t.Errorf("Chunk.messageTimestamps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Chunk.messageTimestamps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupID_AsThreadID(t *testing.T) {
	tests := []struct {
		name          string
		id            GroupID
		wantChannelID string
		wantThreadTS  string
		wantOk        bool
	}{
		{
			name:          "channel",
			id:            "C123",
			wantChannelID: "",
			wantThreadTS:  "",
			wantOk:        false,
		},
		{
			name:          "thread",
			id:            "tC123:1234",
			wantChannelID: "C123",
			wantThreadTS:  "1234",
			wantOk:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChannelID, gotThreadTS, gotOk := tt.id.AsThreadID()
			if gotChannelID != tt.wantChannelID {
				t.Errorf("GroupID.AsThreadID() gotChannelID = %v, want %v", gotChannelID, tt.wantChannelID)
			}
			if gotThreadTS != tt.wantThreadTS {
				t.Errorf("GroupID.AsThreadID() gotThreadTS = %v, want %v", gotThreadTS, tt.wantThreadTS)
			}
			if gotOk != tt.wantOk {
				t.Errorf("GroupID.AsThreadID() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestGroupID_ExtractChannelID(t *testing.T) {
	tests := []struct {
		name          string
		id            GroupID
		wantChannelID string
		wantOk        bool
	}{
		{
			name:          "channel",
			id:            "C123",
			wantChannelID: "C123",
			wantOk:        true,
		},
		{
			name:          "thread",
			id:            "tC123:1234",
			wantChannelID: "C123",
			wantOk:        true,
		},
		{
			name:          "invalid",
			id:            "invalid",
			wantChannelID: "",
			wantOk:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChannelID, gotOk := tt.id.ExtractChannelID()
			if gotChannelID != tt.wantChannelID {
				t.Errorf("GroupID.ExtractChannelID() gotChannelID = %v, want %v", gotChannelID, tt.wantChannelID)
			}
			if gotOk != tt.wantOk {
				t.Errorf("GroupID.ExtractChannelID() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}
