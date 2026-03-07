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
	"iter"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rusq/slack"

	st "github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/source"
)

type aliasSourceStub struct {
	aliases map[string]string
}

func (*aliasSourceStub) Name() string                                      { return "test" }
func (*aliasSourceStub) Type() source.Flags                                { return source.FDatabase }
func (*aliasSourceStub) Channels(context.Context) ([]slack.Channel, error) { return nil, nil }
func (*aliasSourceStub) Users(context.Context) ([]slack.User, error)       { return nil, nil }
func (*aliasSourceStub) AllMessages(context.Context, string) (iter.Seq2[slack.Message, error], error) {
	return nil, nil
}
func (*aliasSourceStub) AllThreadMessages(context.Context, string, string) (iter.Seq2[slack.Message, error], error) {
	return nil, nil
}
func (*aliasSourceStub) Sorted(context.Context, string, bool, func(time.Time, *slack.Message) error) error {
	return nil
}
func (*aliasSourceStub) ChannelInfo(context.Context, string) (*slack.Channel, error) {
	return &slack.Channel{}, nil
}
func (*aliasSourceStub) Files() source.Storage   { return source.NoStorage{} }
func (*aliasSourceStub) Avatars() source.Storage { return source.NoStorage{} }
func (*aliasSourceStub) WorkspaceInfo(context.Context) (*slack.AuthTestResponse, error) {
	return nil, nil
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
