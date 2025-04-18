package chunk

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/rusq/slack"

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

	t.Run("doesn't fail on invalid json", func(t *testing.T) {
		testdir := fstest.MapFS{
			"C123VALID.json.gz": &fstest.MapFile{
				Data: compress(t, marshal(t, channelInfos[0])),
			},
			"C123INVALID.json.gz": &fstest.MapFile{
				Data: compress(t, []byte("invalid json")),
			},
			"C123VALID2.json.gz": &fstest.MapFile{
				Data: compress(t, marshal(t, channelInfos[1])),
			},
		}

		dir := prepDir(t, testdir)
		d, err := OpenDir(dir)
		if err != nil {
			t.Fatalf("OpenDir() error = %v", err)
		}
		defer d.Close()
		var seen []string
		if err := d.Walk(func(name string, f *File, err error) error {
			if err != nil {
				t.Fatalf("Walk() error = %v", err)
			}
			if name == "C123INVALID.json.gz" {
				t.Fatal("should not be called for invalid json")
			}
			if f == nil {
				t.Fatalf("Walk() file is nil")
			}
			seen = append(seen, name)
			return nil
		}); err != nil {
			t.Fatalf("Walk() error = %v", err)
		}
		if len(seen) != 2 {
			t.Fatalf("Walk() = %v, want 2", len(seen))
		}
	})
}
