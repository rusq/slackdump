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

package viewer

import (
	"context"
	"io/fs"
	"iter"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/rusq/slack"

	st "github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/internal/viewer/renderer"
	"github.com/rusq/slackdump/v4/source"
)

type aliasSourceStub struct {
	aliases map[string]string
	name    string
	flags   source.Flags
	chs     []slack.Channel
	users   []slack.User
	msgs    map[string][]slack.Message
	threads map[string]map[string][]slack.Message
	files   source.Storage
	avatars source.Storage
	wi      *slack.AuthTestResponse
}

func (s *aliasSourceStub) Name() string {
	if s.name != "" {
		return s.name
	}
	return "test"
}
func (s *aliasSourceStub) Type() source.Flags {
	if s.flags != 0 {
		return s.flags
	}
	return source.FDatabase
}
func (s *aliasSourceStub) Channels(context.Context) ([]slack.Channel, error) { return s.chs, nil }
func (s *aliasSourceStub) Users(context.Context) ([]slack.User, error)       { return s.users, nil }
func (s *aliasSourceStub) AllMessages(_ context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	if s.msgs == nil {
		return nil, nil
	}
	return messageSeq(s.msgs[channelID]), nil
}
func (s *aliasSourceStub) AllThreadMessages(_ context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	if s.threads == nil {
		return nil, nil
	}
	mm, ok := s.threads[channelID][threadID]
	if !ok {
		return nil, source.ErrNotFound
	}
	return messageSeq(mm), nil
}
func (s *aliasSourceStub) Sorted(_ context.Context, channelID string, _ bool, cb func(time.Time, *slack.Message) error) error {
	for i := range s.msgs[channelID] {
		if err := cb(time.Time{}, &s.msgs[channelID][i]); err != nil {
			return err
		}
	}
	for _, mm := range s.threads[channelID] {
		for i := range mm {
			if err := cb(time.Time{}, &mm[i]); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s *aliasSourceStub) ChannelInfo(_ context.Context, channelID string) (*slack.Channel, error) {
	for _, ch := range s.chs {
		if ch.ID == channelID {
			copy := ch
			return &copy, nil
		}
	}
	return &slack.Channel{}, nil
}
func (s *aliasSourceStub) Files() source.Storage {
	if s.files != nil {
		return s.files
	}
	return source.NoStorage{}
}
func (s *aliasSourceStub) Avatars() source.Storage {
	if s.avatars != nil {
		return s.avatars
	}
	return source.NoStorage{}
}
func (s *aliasSourceStub) WorkspaceInfo(context.Context) (*slack.AuthTestResponse, error) {
	return s.wi, nil
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

type storageStub struct {
	fsys       fs.FS
	byName     map[string]string
	byID       map[string]string
	allowByID  bool
	storageTyp source.StorageType
}

func (s storageStub) FS() fs.FS {
	if s.fsys != nil {
		return s.fsys
	}
	return fstest.MapFS{}
}

func (s storageStub) Type() source.StorageType {
	if s.storageTyp != 0 {
		return s.storageTyp
	}
	return source.STmattermost
}

func (s storageStub) File(id, name string) (string, error) {
	if p, ok := s.byName[id+"/"+name]; ok {
		return p, nil
	}
	return "", fs.ErrNotExist
}

func (s storageStub) FileByID(id string) (string, error) {
	if !s.allowByID {
		return "", fs.ErrNotExist
	}
	if p, ok := s.byID[id]; ok {
		return p, nil
	}
	return "", fs.ErrNotExist
}

func (s storageStub) FilePath(_ *slack.Channel, f *slack.File) string {
	return path.Join(f.ID, f.Name)
}

func newViewerRouteSource() *aliasSourceStub {
	channel := slack.Channel{
		GroupConversation: slack.GroupConversation{
			Name:         "general",
			Conversation: slack.Conversation{ID: "C1"},
			Topic:        slack.Topic{Value: "General discussion"},
		},
		Properties: &slack.Properties{Canvas: slack.Canvas{FileId: "FCANVAS"}},
		IsChannel:  true,
	}
	return &aliasSourceStub{
		chs: []slack.Channel{channel},
		users: []slack.User{{
			ID:      "U1",
			Profile: slack.UserProfile{RealName: "Ada Lovelace", Image512: "https://example.com/avatar.png"},
		}},
		msgs: map[string][]slack.Message{
			"C1": {
				{Msg: slack.Msg{Timestamp: "1710000000.000001", ThreadTimestamp: "1710000000.000001", LatestReply: "1710000001.000001", ReplyCount: 1, User: "U1", Text: "thread root", Files: []slack.File{{ID: "F1", Name: "hello.txt"}}}},
			},
		},
		threads: map[string]map[string][]slack.Message{
			"C1": {
				"1710000000.000001": {
					{Msg: slack.Msg{Timestamp: "1710000000.000001", ThreadTimestamp: "1710000000.000001", LatestReply: "1710000001.000001", ReplyCount: 1, User: "U1", Text: "thread root"}},
					{Msg: slack.Msg{Timestamp: "1710000001.000001", ThreadTimestamp: "1710000000.000001", User: "U1", Text: "reply body"}},
				},
			},
		},
		files: storageStub{
			fsys: fstest.MapFS{
				"F1/hello.txt":        {Data: []byte("hello")},
				"FCANVAS/canvas.html": {Data: []byte("<html><body>canvas body</body></html>")},
			},
			byName:    map[string]string{"F1/hello.txt": "F1/hello.txt"},
			byID:      map[string]string{"FCANVAS": "FCANVAS/canvas.html"},
			allowByID: true,
		},
	}
}

func (s *aliasSourceStub) Alias(id string) (string, bool, error) {
	v, ok := s.aliases[id]
	return v, ok, nil
}

func (s *aliasSourceStub) SetAlias(id, alias string) error {
	if s.aliases == nil {
		s.aliases = map[string]string{}
	}
	s.aliases[id] = alias
	return nil
}

func (s *aliasSourceStub) DeleteAlias(id string) error {
	delete(s.aliases, id)
	return nil
}

func (s *aliasSourceStub) Aliases() (map[string]string, error) {
	if s.aliases == nil {
		return map[string]string{}, nil
	}
	return s.aliases, nil
}

func TestValidateAlias(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		action  aliasAction
		wantErr bool
	}{
		{name: "set simple", in: "alpha_1", want: "alpha_1", action: aliasSet},
		{name: "trim spaces", in: "  alpha-1  ", want: "alpha-1", action: aliasSet},
		{name: "empty means delete", in: "   ", want: "", action: aliasDelete},
		{name: "unicode letters allowed", in: "абв", want: "абв", action: aliasSet},
		{name: "invalid char", in: "bad alias", wantErr: true},
		{name: "too long", in: strings.Repeat("a", 31), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, action, err := validateAlias(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateAlias() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("validateAlias() alias = %q, want %q", got, tt.want)
			}
			if action != tt.action {
				t.Fatalf("validateAlias() action = %v, want %v", action, tt.action)
			}
		})
	}
}

func TestChannelDisplayName(t *testing.T) {
	v := &Viewer{
		um:  st.UserIndex{},
		src: &aliasSourceStub{aliases: map[string]string{"C1": "alpha"}},
	}
	ch := slack.Channel{
		GroupConversation: slack.GroupConversation{
			Name: "general",
			Conversation: slack.Conversation{
				ID: "C1",
			},
		},
		IsChannel: true,
	}
	got := string(v.channelDisplayName(ch))
	if got != "#<em>alpha</em>" {
		t.Fatalf("channelDisplayName() = %q, want %q", got, "#<em>alpha</em>")
	}
}

func TestAliasPutHandler(t *testing.T) {
	src := &aliasSourceStub{}
	v := &Viewer{
		ch: channels{Public: []slack.Channel{{
			GroupConversation: slack.GroupConversation{
				Name: "general",
				Conversation: slack.Conversation{
					ID: "C1",
				},
			},
			IsChannel: true,
		}}},
		um:  st.UserIndex{},
		src: src,
		lg:  slog.Default(),
	}
	initTemplates(v)

	req := httptest.NewRequest(http.MethodPut, "/archives/C1/alias/", strings.NewReader("alias=alpha"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "C1")
	rr := httptest.NewRecorder()

	v.aliasPutHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("aliasPutHandler() status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := src.aliases["C1"]; got != "alpha" {
		t.Fatalf("aliasPutHandler() alias = %q, want %q", got, "alpha")
	}
	if !strings.Contains(rr.Body.String(), "<em>alpha</em>") {
		t.Fatalf("aliasPutHandler() response = %q, want italic alias", rr.Body.String())
	}
}

func TestAliasPutHandlerInvalid(t *testing.T) {
	src := &aliasSourceStub{}
	v := &Viewer{
		ch: channels{Public: []slack.Channel{{
			GroupConversation: slack.GroupConversation{
				Name: "general",
				Conversation: slack.Conversation{
					ID: "C1",
				},
			},
			IsChannel: true,
		}}},
		um:  st.UserIndex{},
		src: src,
		lg:  slog.Default(),
	}
	initTemplates(v)

	req := httptest.NewRequest(http.MethodPut, "/archives/C1/alias/", strings.NewReader("alias=bad alias"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "C1")
	rr := httptest.NewRecorder()

	v.aliasPutHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("aliasPutHandler() invalid status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if _, ok := src.aliases["C1"]; ok {
		t.Fatalf("aliasPutHandler() should not persist invalid alias")
	}
	if !strings.Contains(rr.Body.String(), "letters, digits, underscores, and dashes") {
		t.Fatalf("aliasPutHandler() response = %q, want validation error", rr.Body.String())
	}
}

func TestAliasPutHandler_Lifecycle(t *testing.T) {
	src := &aliasSourceStub{}
	v := &Viewer{
		ch: channels{Public: []slack.Channel{{
			GroupConversation: slack.GroupConversation{
				Name: "general",
				Conversation: slack.Conversation{
					ID: "C1",
				},
			},
			IsChannel: true,
		}}},
		um:  st.UserIndex{},
		src: src,
		lg:  slog.Default(),
		rts: renderer.NewRoutes(renderer.ModeLive),
	}
	initTemplates(v)

	for _, tc := range []struct {
		name       string
		alias      string
		wantAlias  string
		wantBody   string
		bodyAbsent string
	}{
		{name: "set", alias: "alpha", wantAlias: "alpha", wantBody: "<em>alpha</em>"},
		{name: "update", alias: "beta", wantAlias: "beta", wantBody: "<em>beta</em>", bodyAbsent: "<em>alpha</em>"},
		{name: "delete", alias: "   ", wantAlias: "", wantBody: "#general", bodyAbsent: "<em>beta</em>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/archives/C1/alias/", strings.NewReader("alias="+tc.alias))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.SetPathValue("id", "C1")
			rr := httptest.NewRecorder()

			v.aliasPutHandler(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("aliasPutHandler() status = %d, want %d", rr.Code, http.StatusOK)
			}
			gotAlias, _, _ := src.Alias("C1")
			if gotAlias != tc.wantAlias {
				t.Fatalf("Alias() = %q, want %q", gotAlias, tc.wantAlias)
			}
			body := rr.Body.String()
			if !strings.Contains(body, tc.wantBody) {
				t.Fatalf("aliasPutHandler() body = %q, want substring %q", body, tc.wantBody)
			}
			if tc.bodyAbsent != "" && strings.Contains(body, tc.bodyAbsent) {
				t.Fatalf("aliasPutHandler() body = %q, should not contain %q", body, tc.bodyAbsent)
			}
		})
	}
}
