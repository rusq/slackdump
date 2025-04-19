package chunk

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/testutil"
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
				seen = append(seen, strings.TrimPrefix(name, dir))
				return nil
			}); (err != nil) != tt.wantErr {
				t.Fatalf("Walk() wantErr: %v, got error = %v", tt.wantErr, err)
			}
			assert.ElementsMatch(t, tt.want, seen)
		})
	}
}
