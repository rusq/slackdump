package chunk

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/fixtures"
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

func TestOpenDir(t *testing.T) {
}

func TestDirectory_Walk(t *testing.T) {
	var (
		// prepDir prepares a temporary directory for testing and populates it with
		// files from fsys.  It returns the path to the directory.
		prepDir = func(t *testing.T, fsys fs.FS) string {
			t.Helper()
			dir := t.TempDir()
			if err := os.CopyFS(dir, fsys); err != nil {
				t.Fatal(err)
			}
			return dir
		}

		compress = func(t *testing.T, data []byte) []byte {
			t.Helper()
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			defer gz.Close()
			if _, err := gz.Write(data); err != nil {
				t.Fatal(err)
			}
			if err := gz.Close(); err != nil {
				t.Fatal(err)
			}
			return buf.Bytes()
		}

		marshal = func(t *testing.T, v any) []byte {
			t.Helper()
			data, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			return data
		}
	)

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
					Data: compress(t, marshal(t, channelInfos[0])),
				},
				"C123INVALID.json.gz": &fstest.MapFile{
					Data: compress(t, []byte("invalid json")),
				},
				"C123VALID2.json.gz": &fstest.MapFile{
					Data: compress(t, marshal(t, channelInfos[1])),
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
					Data: compress(t, []byte("NaN")),
				},
				"__uploads/CVALID.json.gz": &fstest.MapFile{
					Data: compress(t, marshal(t, channelInfos[1])),
				},
				"__avatars/CVALID.json.gz": &fstest.MapFile{
					Data: compress(t, marshal(t, channelInfos[2])),
				},
				"somedir/CVALID.json.gz": &fstest.MapFile{
					Data: compress(t, marshal(t, channelInfos[3])),
				},
				"CVALID.json.gz": &fstest.MapFile{
					Data: compress(t, marshal(t, channelInfos[0])),
				},
				"CANOTHER.json.gz": &fstest.MapFile{
					Data: compress(t, marshal(t, channelInfos[1])),
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
			dir := prepDir(t, tt.fsys)
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
				seen = append(seen, strings.TrimLeft(name, dir))
				return nil
			}); (err != nil) != tt.wantErr {
				t.Fatalf("Walk() wantErr: %v, got error = %v", tt.wantErr, err)
			}
			assert.ElementsMatch(t, tt.want, seen)
		})
	}
}
