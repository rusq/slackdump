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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rusq/slack"

	st "github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/internal/viewer/renderer"
)

func Test_isInvalid(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"relative path", args{"../test.txt"}, true},
		{"home dir ref", args{"~/test.txt"}, true},
		{"filename with tilda #561", args{"test~1.txt"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInvalid(tt.args.path); got != tt.want {
				t.Errorf("isInvalid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserHandler_RendersFullPageWithoutHTMX(t *testing.T) {
	v := &Viewer{
		src: &aliasSourceStub{},
		um:  st.UserIndex{"U1": &slack.User{ID: "U1", Profile: slack.UserProfile{RealName: "Ada"}}},
		lg:  slog.Default(),
		r:   &renderer.Debug{},
		rts: renderer.NewRoutes(renderer.ModeLive),
	}
	initTemplates(v)

	req := httptest.NewRequest(http.MethodGet, "/team/U1", nil)
	req.SetPathValue("user_id", "U1")
	rr := httptest.NewRecorder()

	v.userHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("userHandler() status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("userHandler() response should include full page HTML: %q", body)
	}
	if !strings.Contains(body, "Profile") {
		t.Fatalf("userHandler() response should include profile panel: %q", body)
	}
}

func TestUserHandler_RendersHTMXUserPanel(t *testing.T) {
	v := &Viewer{
		src: &aliasSourceStub{},
		um: st.UserIndex{"U1": &slack.User{ID: "U1", Profile: slack.UserProfile{
			RealName: "Ada Lovelace",
			Image512: "https://example.com/avatar.png",
		}}},
		lg:  slog.Default(),
		r:   &renderer.Debug{},
		rts: renderer.NewRoutes(renderer.ModeLive),
	}
	initTemplates(v)

	req := httptest.NewRequest(http.MethodGet, "/team/U1", nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("user_id", "U1")
	rr := httptest.NewRecorder()

	v.userHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("userHandler() status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("userHandler() HTMX response should not include full page HTML: %q", body)
	}
	if !strings.Contains(body, "Ada Lovelace") {
		t.Fatalf("userHandler() HTMX response should include user details: %q", body)
	}
	if !strings.Contains(body, `id="close-user"`) {
		t.Fatalf("userHandler() HTMX response should include close button: %q", body)
	}
	if strings.Contains(body, "Unknown") {
		t.Fatalf("userHandler() HTMX response should not render unknown user: %q", body)
	}
}
