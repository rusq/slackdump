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
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/internal/testutil"
)

// assortment of channel info chunks
var (
	TestPublicChannelInfo = Chunk{
		ChannelID: "C01SPFM1KNY",
		Type:      CChannelInfo,
		Channel: &slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:       "C01SPFM1KNY",
					IsShared: false,
				},
				Name:       "test",
				IsArchived: false,
			},
			IsChannel: true,
			IsMember:  true,
			IsGeneral: false,
		},
	}
	TestDMChannelInfo = Chunk{
		ChannelID: "D01MN4X7UGP",
		Type:      CChannelInfo,
		Channel: &slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:          "D01MN4X7UGP",
					IsOpen:      true,
					IsIM:        true,
					IsPrivate:   true,
					IsOrgShared: false,
				},
			},
		},
	}
	TestChannelUsers = Chunk{
		ChannelID: "C01SPFM1KNY",
		Type:      CChannelUsers,
		ChannelUsers: []string{
			"U01SPFM1KNY",
			"U01SPFM1KNZ",
			"U01SPFM1KNA",
		},
	}
)

// assortment of message chunks
var (
	TestPublicChannelMessages = Chunk{
		Type:      CMessages,
		ChannelID: "C01SPFM1KNY",
		Messages: []slack.Message{
			fixtures.Load[slack.Message](fixtures.TestMessage),
		},
	}
)

func TestDirectory_Walk(t *testing.T) {
	fixtures.SkipOnWindows(t) // TODO: fix this test on Windows
	testChannels := fixtures.Load[[]slack.Channel](fixtures.TestChannelsJSON)
	channelInfos := make([]Chunk, len(testChannels))
	for _, ch := range testChannels {
		channelInfos = append(channelInfos, Chunk{
			Type:      CChannelInfo,
			ChannelID: ch.ID,
			Channel:   &ch,
		})
	}
	if len(channelInfos) == 0 {
		t.Fatal("fixture has no channels")
	}

	tests := []struct {
		name    string
		fsys    fs.FS
		want    []string
		wantErr bool
	}{
		{
			name: "invalid json in root",
			fsys: fstest.MapFS{
				"C123VALID.json.gz": &fstest.MapFile{
					Data: testutil.GZCompress(t, testutil.MarshalJSON(t, channelInfos[0])),
				},
				"C123INVALID.json.gz": &fstest.MapFile{
					Data: testutil.GZCompress(t, []byte("invalid json")),
				},
				"C123VALID2.json.gz": &fstest.MapFile{
					Data: testutil.GZCompress(t, testutil.MarshalJSON(t, channelInfos[1])),
				},
			},
			want: []string{
				"C123VALID.json.gz",
				"C123VALID2.json.gz",
			},
		},
		{
			name: "should scan only top level dir",
			fsys: fstest.MapFS{
				"__uploads/CINVALID.json.gz": &fstest.MapFile{
					Data: testutil.GZCompress(t, []byte("NaN")),
				},
				"__uploads/CVALID.json.gz": &fstest.MapFile{
					Data: testutil.GZCompress(t, testutil.MarshalJSON(t, channelInfos[1])),
				},
				"__avatars/CVALID.json.gz": &fstest.MapFile{
					Data: testutil.GZCompress(t, testutil.MarshalJSON(t, channelInfos[2])),
				},
				"somedir/CVALID.json.gz": &fstest.MapFile{
					Data: testutil.GZCompress(t, testutil.MarshalJSON(t, channelInfos[3])),
				},
				"CVALID.json.gz": &fstest.MapFile{
					Data: testutil.GZCompress(t, testutil.MarshalJSON(t, channelInfos[0])),
				},
				"CANOTHER.json.gz": &fstest.MapFile{
					Data: testutil.GZCompress(t, testutil.MarshalJSON(t, channelInfos[1])),
				},
			},
			want: []string{
				"CVALID.json.gz",
				"CANOTHER.json.gz",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := testutil.PrepareTestDirectory(t, tt.fsys)
			d, err := OpenDir(dir)
			if err != nil {
				t.Fatalf("OpenDir() error = %v", err)
			}
			defer d.Close()
			var seen []string
			if err := d.Walk(func(name string, f *File, err error) error {
				if err != nil {
					return err
				}
				if f == nil {
					return errors.New("file is nil")
				}
				t.Logf("name: %q, trimmed: %q", name, strings.TrimLeft(name, dir))
				seen = append(seen, strings.TrimLeft(name, dir))
				return nil
			}); (err != nil) != tt.wantErr {
				t.Fatalf("Walk() wantErr: %v, got error = %v", tt.wantErr, err)
			}
			assert.ElementsMatch(t, tt.want, seen)
		})
	}
}
