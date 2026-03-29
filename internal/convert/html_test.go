package convert

import (
	"context"
	"errors"
	"io/fs"
	"iter"
	"os"
	"path"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/source"
)

func TestHTMLConverter_Convert(t *testing.T) {
	t.Run("copies pages and local assets", func(t *testing.T) {
		src := &htmlSourceStub{
			channels: []slack.Channel{
				{
					GroupConversation: slack.GroupConversation{
						Name:         "general",
						Conversation: slack.Conversation{ID: "C1"},
						Topic:        slack.Topic{Value: "General discussion"},
					},
					Properties: &slack.Properties{Canvas: slack.Canvas{FileId: "Fcanvas"}},
					IsChannel:  true,
				},
				{
					GroupConversation: slack.GroupConversation{
						Name:         "empty",
						Conversation: slack.Conversation{ID: "CEMPTY"},
					},
					IsChannel: true,
				},
			},
			users: []slack.User{{ID: "U1", Profile: slack.UserProfile{RealName: "Ada Lovelace", ImageOriginal: "https://example.com/ada.png", Image48: "https://example.com/ada.png", Image512: "https://example.com/ada.png"}}},
			messages: map[string][]slack.Message{
				"C1": {
					{Msg: slack.Msg{Timestamp: "1710000000.000001", ThreadTimestamp: "1710000000.000001", LatestReply: "1710000002.000001", ReplyCount: 1, User: "U1", Text: "thread root", Files: []slack.File{{ID: "F1", Name: "hello.txt"}}}},
					{Msg: slack.Msg{Timestamp: "1710000005.000001", User: "U1", Text: "plain message"}},
				},
			},
			threads: map[string]map[string][]slack.Message{
				"C1": {
					"1710000000.000001": {
						{Msg: slack.Msg{Timestamp: "1710000000.000001", ThreadTimestamp: "1710000000.000001", LatestReply: "1710000002.000001", ReplyCount: 1, User: "U1", Text: "thread root"}},
						{Msg: slack.Msg{Timestamp: "1710000002.000001", ThreadTimestamp: "1710000000.000001", User: "U1", Text: "reply"}},
					},
				},
			},
			files: htmlStorage{
				fsys: fstest.MapFS{
					"F1/hello.txt":        {Data: []byte("hello")},
					"Fcanvas/canvas.html": {Data: []byte("<html><body>canvas</body></html>")},
				},
				byID: map[string]string{"F1": "F1/hello.txt", "Fcanvas": "Fcanvas/canvas.html"},
			},
			avatars: htmlStorage{
				fsys: fstest.MapFS{
					"U1/ada.png": {Data: []byte("avatar")},
				},
				byID: map[string]string{},
			},
		}

		outDir := t.TempDir()
		conv := NewToHTML(src, fsadapter.NewDirectory(outDir))
		if err := conv.Convert(t.Context()); err != nil {
			fatalTree(t, outDir)
			t.Fatalf("Convert() error = %v", err)
		}

		for _, name := range []string{
			"index.html",
			"archives/C1/index.html",
			"archives/C1/threads/1710000000.000001.html",
			"archives/C1/canvas/index.html",
			"archives/C1/canvas/content.html",
			"archives/CEMPTY/index.html",
			"files/F1/hello.txt",
			"avatars/U1/ada.png",
			"static/48x48.gif",
			"team/U1/index.html",
		} {
			if _, err := fs.Stat(osDirFS(outDir), name); err != nil {
				t.Fatalf("missing output %q: %v", name, err)
			}
		}

		channelBody := readFile(t, outDir, "archives/C1/index.html")
		if !strings.Contains(channelBody, "<!DOCTYPE html>") {
			t.Fatalf("channel page should be full HTML: %q", channelBody)
		}
		if strings.Contains(channelBody, `hx-get`) {
			t.Fatalf("channel page should not include HTMX attributes: %q", channelBody)
		}
		if !strings.Contains(channelBody, `href="../../team/U1/index.html"`) {
			t.Fatalf("channel page should rewrite user links relatively: %q", channelBody)
		}
		if !strings.Contains(channelBody, `href="../../files/F1/hello.txt"`) {
			t.Fatalf("channel page should rewrite file links relatively: %q", channelBody)
		}

		indexBody := readFile(t, outDir, "index.html")
		if !strings.Contains(indexBody, `href="archives/C1/index.html"`) {
			t.Fatalf("index page should contain relative channel link: %q", indexBody)
		}

		threadBody := readFile(t, outDir, "archives/C1/threads/1710000000.000001.html")
		if !strings.Contains(threadBody, `href="../../../team/U1/index.html"`) {
			t.Fatalf("thread page should rewrite user links relatively: %q", threadBody)
		}

		canvasBody := readFile(t, outDir, "archives/C1/canvas/content.html")
		if !strings.Contains(canvasBody, "canvas") {
			t.Fatalf("canvas content should be written, got %q", canvasBody)
		}

		userBody := readFile(t, outDir, "team/U1/index.html")
		if !strings.Contains(userBody, `src="../../avatars/U1/ada.png"`) {
			t.Fatalf("user page should rewrite avatar links relatively: %q", userBody)
		}
	})

	t.Run("missing assets do not fail", func(t *testing.T) {
		src := &htmlSourceStub{
			channels: []slack.Channel{{
				GroupConversation: slack.GroupConversation{
					Name:         "general",
					Conversation: slack.Conversation{ID: "C1"},
				},
				IsChannel: true,
			}},
			users: []slack.User{{ID: "U1", Profile: slack.UserProfile{RealName: "Ada Lovelace", ImageOriginal: "https://example.com/ada.png", Image512: "https://example.com/ada.png"}}},
			messages: map[string][]slack.Message{
				"C1": {
					{Msg: slack.Msg{Timestamp: "1710000000.000001", User: "U1", Text: "plain message", Files: []slack.File{{ID: "Fmissing", Name: "missing.txt"}}}},
				},
			},
			files: htmlStorage{
				fsys: fstest.MapFS{},
				byID: map[string]string{"Fmissing": "Fmissing/missing.txt"},
			},
			avatars: htmlStorage{
				fsys: fstest.MapFS{},
			},
		}

		outDir := t.TempDir()
		conv := NewToHTML(src, fsadapter.NewDirectory(outDir))
		if err := conv.Convert(t.Context()); err != nil {
			fatalTree(t, outDir)
			t.Fatalf("Convert() error = %v", err)
		}
		if _, err := fs.Stat(osDirFS(outDir), "files/Fmissing/missing.txt"); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("missing file asset should be skipped, got err=%v", err)
		}
	})
}

func TestRelativePrefix(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "index.html", want: ""},
		{path: "archives/C1/index.html", want: "../../"},
		{path: "archives/C1/threads/1.html", want: "../../../"},
	}
	for _, tt := range tests {
		if got := relativePrefix(tt.path); got != tt.want {
			t.Fatalf("relativePrefix(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

type htmlSourceStub struct {
	channels []slack.Channel
	users    []slack.User
	messages map[string][]slack.Message
	threads  map[string]map[string][]slack.Message
	files    source.Storage
	avatars  source.Storage
}

func (*htmlSourceStub) Name() string       { return "test-archive" }
func (*htmlSourceStub) Type() source.Flags { return source.FChunk | source.FDirectory }
func (s *htmlSourceStub) Channels(context.Context) ([]slack.Channel, error) {
	return s.channels, nil
}
func (s *htmlSourceStub) Users(context.Context) ([]slack.User, error) { return s.users, nil }
func (s *htmlSourceStub) AllMessages(_ context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	mm, ok := s.messages[channelID]
	if !ok {
		return nil, source.ErrNotFound
	}
	return messageSeq(mm), nil
}
func (s *htmlSourceStub) AllThreadMessages(_ context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	threads, ok := s.threads[channelID]
	if !ok {
		return nil, source.ErrNotFound
	}
	mm, ok := threads[threadID]
	if !ok {
		return nil, source.ErrNotFound
	}
	return messageSeq(mm), nil
}
func (s *htmlSourceStub) Sorted(_ context.Context, channelID string, _ bool, cb func(time.Time, *slack.Message) error) error {
	if mm, ok := s.messages[channelID]; ok {
		for i := range mm {
			if err := cb(time.Time{}, &mm[i]); err != nil {
				return err
			}
		}
	}
	if threads, ok := s.threads[channelID]; ok {
		for _, mm := range threads {
			for i := range mm {
				if err := cb(time.Time{}, &mm[i]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func (s *htmlSourceStub) ChannelInfo(_ context.Context, channelID string) (*slack.Channel, error) {
	for _, ch := range s.channels {
		if ch.ID == channelID {
			copy := ch
			return &copy, nil
		}
	}
	return nil, source.ErrNotFound
}
func (s *htmlSourceStub) Files() source.Storage { return s.files }
func (s *htmlSourceStub) Avatars() source.Storage {
	if s.avatars == nil {
		return source.NoStorage{}
	}
	return s.avatars
}
func (*htmlSourceStub) WorkspaceInfo(context.Context) (*slack.AuthTestResponse, error) {
	return &slack.AuthTestResponse{URL: "https://example.slack.com"}, nil
}

type htmlStorage struct {
	fsys fs.FS
	byID map[string]string
}

func (s htmlStorage) FS() fs.FS { return s.fsys }
func (s htmlStorage) Type() source.StorageType {
	if len(s.byID) == 0 && s.fsys == nil {
		return source.STnone
	}
	return source.STdump
}
func (s htmlStorage) File(id, name string) (string, error) {
	if p, ok := s.byID[id]; ok {
		return p, nil
	}
	p := path.Join(id, name)
	if _, err := fs.Stat(s.fsys, p); err != nil {
		return "", err
	}
	return p, nil
}
func (s htmlStorage) FileByID(id string) (string, error) {
	p, ok := s.byID[id]
	if !ok {
		return "", fs.ErrNotExist
	}
	return p, nil
}
func (s htmlStorage) FilePath(_ *slack.Channel, f *slack.File) string {
	return path.Join(f.ID, f.Name)
}

func osDirFS(root string) fs.FS { return os.DirFS(root) }

func readFile(t *testing.T, root, name string) string {
	t.Helper()
	b, err := fs.ReadFile(osDirFS(root), name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func fatalTree(t *testing.T, root string) {
	t.Helper()
	_ = fs.WalkDir(osDirFS(root), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Logf("walk error: %v", err)
			return nil
		}
		t.Log(name)
		return nil
	})
}

func messageSeq(mm []slack.Message) iter.Seq2[slack.Message, error] {
	return func(yield func(slack.Message, error) bool) {
		for _, msg := range mm {
			if !yield(msg, nil) {
				return
			}
		}
	}
}
