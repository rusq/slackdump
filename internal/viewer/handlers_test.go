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
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rusq/slack"

	st "github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/internal/viewer/renderer"
	"github.com/rusq/slackdump/v4/source"
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

func newHandlerTestViewer(src *aliasSourceStub) *Viewer {
	v := &Viewer{
		src: src,
		ch:  initChannels(src.chs),
		um:  st.NewUserIndex(src.users),
		lg:  slog.Default(),
		r:   &renderer.Debug{},
		rts: renderer.NewRoutes(renderer.ModeLive),
	}
	initTemplates(v)
	return v
}

func TestChannelHandler_RendersFullPageWithoutHTMX(t *testing.T) {
	v := newHandlerTestViewer(newViewerRouteSource())
	req := httptest.NewRequest(http.MethodGet, "/archives/C1", nil)
	rr := httptest.NewRecorder()

	v.channelHandler(rr, req, "C1")

	if rr.Code != http.StatusOK {
		t.Fatalf("channelHandler() status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("channelHandler() should render full page HTML: %q", body)
	}
	if !strings.Contains(body, `hx-get="/archives/C1"`) {
		t.Fatalf("channelHandler() full page should preserve live HTMX routes: %q", body)
	}
	if !strings.Contains(body, "thread root") {
		t.Fatalf("channelHandler() should render channel messages: %q", body)
	}
}

func TestChannelHandler_RendersHTMXPartial(t *testing.T) {
	v := newHandlerTestViewer(newViewerRouteSource())
	req := httptest.NewRequest(http.MethodGet, "/archives/C1", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	v.channelHandler(rr, req, "C1")

	if rr.Code != http.StatusOK {
		t.Fatalf("channelHandler() status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("channelHandler() HTMX response should not include full page HTML: %q", body)
	}
	if !strings.Contains(body, `id="tab-panel-conversation"`) {
		t.Fatalf("channelHandler() HTMX response should include conversation panel: %q", body)
	}
	if !strings.Contains(body, "thread root") {
		t.Fatalf("channelHandler() HTMX response should include message content: %q", body)
	}
}

func TestThreadHandler_RendersFullPageWithoutHTMX(t *testing.T) {
	v := newHandlerTestViewer(newViewerRouteSource())
	req := httptest.NewRequest(http.MethodGet, "/archives/C1/1710000000.000001", nil)
	req.SetPathValue("ts", "1710000000.000001")
	rr := httptest.NewRecorder()

	v.threadHandler(rr, req, "C1")

	if rr.Code != http.StatusOK {
		t.Fatalf("threadHandler() status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("threadHandler() should render full page HTML: %q", body)
	}
	if !strings.Contains(body, "Link to this thread") {
		t.Fatalf("threadHandler() should include thread panel: %q", body)
	}
	if !strings.Contains(body, "thread root") || !strings.Contains(body, "reply body") {
		t.Fatalf("threadHandler() should render thread messages: %q", body)
	}
}

func TestThreadHandler_RendersHTMXPartial(t *testing.T) {
	v := newHandlerTestViewer(newViewerRouteSource())
	req := httptest.NewRequest(http.MethodGet, "/archives/C1/1710000000.000001", nil)
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("ts", "1710000000.000001")
	rr := httptest.NewRecorder()

	v.threadHandler(rr, req, "C1")

	if rr.Code != http.StatusOK {
		t.Fatalf("threadHandler() status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("threadHandler() HTMX response should not include full page HTML: %q", body)
	}
	if !strings.Contains(body, "Link to this thread") {
		t.Fatalf("threadHandler() HTMX response should include thread body: %q", body)
	}
	if !strings.Contains(body, `id="close-thread"`) {
		t.Fatalf("threadHandler() HTMX response should include close affordance: %q", body)
	}
}

func TestCanvasHandler_RendersFullPageWithoutHTMX(t *testing.T) {
	v := newHandlerTestViewer(newViewerRouteSource())
	req := httptest.NewRequest(http.MethodGet, "/archives/C1/canvas", nil)
	rr := httptest.NewRecorder()

	v.canvasHandler(rr, req, "C1")

	if rr.Code != http.StatusOK {
		t.Fatalf("canvasHandler() status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("canvasHandler() should render full page HTML: %q", body)
	}
	if !strings.Contains(body, `src="/archives/C1/canvas/content"`) {
		t.Fatalf("canvasHandler() should include canvas iframe: %q", body)
	}
}

func TestCanvasHandler_RendersHTMXPartial(t *testing.T) {
	v := newHandlerTestViewer(newViewerRouteSource())
	req := httptest.NewRequest(http.MethodGet, "/archives/C1/canvas", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	v.canvasHandler(rr, req, "C1")

	if rr.Code != http.StatusOK {
		t.Fatalf("canvasHandler() status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("canvasHandler() HTMX response should not include full page HTML: %q", body)
	}
	if !strings.Contains(body, `id="tab-panel-canvas"`) {
		t.Fatalf("canvasHandler() HTMX response should include canvas panel: %q", body)
	}
	if !strings.Contains(body, `sandbox="allow-same-origin"`) {
		t.Fatalf("canvasHandler() HTMX response should preserve iframe sandbox: %q", body)
	}
}

func TestCanvasContentHandler_ServesCanvasHTML(t *testing.T) {
	v := newHandlerTestViewer(newViewerRouteSource())
	req := httptest.NewRequest(http.MethodGet, "/archives/C1/canvas/content", nil)
	rr := httptest.NewRecorder()

	v.canvasContentHandler(rr, req, "C1")

	if rr.Code != http.StatusOK {
		t.Fatalf("canvasContentHandler() status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("canvasContentHandler() content type = %q", got)
	}
	if !strings.Contains(rr.Body.String(), "canvas body") {
		t.Fatalf("canvasContentHandler() should stream canvas HTML: %q", rr.Body.String())
	}
}

func TestCanvasContentHandler_DegradesGracefullyWithoutFileByID(t *testing.T) {
	src := newViewerRouteSource()
	src.files = storageStub{
		fsys:      fstest.MapFS{},
		byName:    map[string]string{},
		byID:      map[string]string{},
		allowByID: false,
	}
	v := newHandlerTestViewer(src)
	req := httptest.NewRequest(http.MethodGet, "/archives/C1/canvas/content", nil)
	rr := httptest.NewRecorder()

	v.canvasContentHandler(rr, req, "C1")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("canvasContentHandler() status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestFileHandler_ServesDownloadedFile(t *testing.T) {
	v := newHandlerTestViewer(newViewerRouteSource())
	req := httptest.NewRequest(http.MethodGet, "/slackdump/file/F1/hello.txt", nil)
	req.SetPathValue("id", "F1")
	req.SetPathValue("filename", "hello.txt")
	rr := httptest.NewRecorder()

	v.fileHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("fileHandler() status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("fileHandler() content type = %q", got)
	}
	if rr.Body.String() != "hello" {
		t.Fatalf("fileHandler() body = %q, want %q", rr.Body.String(), "hello")
	}
}

func TestFileHandler_RejectsInvalidPath(t *testing.T) {
	v := newHandlerTestViewer(newViewerRouteSource())
	req := httptest.NewRequest(http.MethodGet, "/slackdump/file/F1/../hello.txt", nil)
	req.SetPathValue("id", "F1")
	req.SetPathValue("filename", "../hello.txt")
	rr := httptest.NewRecorder()

	v.fileHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("fileHandler() status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestFileHandler_ReturnsNotFoundForMissingFile(t *testing.T) {
	v := newHandlerTestViewer(newViewerRouteSource())
	req := httptest.NewRequest(http.MethodGet, "/slackdump/file/F1/missing.txt", nil)
	req.SetPathValue("id", "F1")
	req.SetPathValue("filename", "missing.txt")
	rr := httptest.NewRecorder()

	v.fileHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("fileHandler() status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestRenderCanvasContent_MissingCanvasReturnsNotExist(t *testing.T) {
	v := newHandlerTestViewer(&aliasSourceStub{
		chs: []slack.Channel{{
			GroupConversation: slack.GroupConversation{
				Name:         "general",
				Conversation: slack.Conversation{ID: "C1"},
			},
			IsChannel: true,
		}},
		files: source.NoStorage{},
	})

	err := v.RenderCanvasContent(t.Context(), "C1", httptest.NewRecorder())
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("RenderCanvasContent() error = %v, want fs.ErrNotExist", err)
	}
}
