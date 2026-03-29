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
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/rusq/slack"

	st "github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/internal/viewer/renderer"
)

func newTestViewer(mode renderer.Mode) *Viewer {
	v := &Viewer{
		src: &aliasSourceStub{},
		um:  st.UserIndex{"U1": &slack.User{ID: "U1", Profile: slack.UserProfile{RealName: "Ada"}}},
		lg:  slog.Default(),
		r:   &renderer.Debug{},
		rts: renderer.NewRoutes(mode),
	}
	initTemplates(v)
	return v
}

func TestRenderIndex(t *testing.T) {
	v := newTestViewer(renderer.ModeLive)
	var buf bytes.Buffer
	if err := v.RenderIndex(context.Background(), &buf); err != nil {
		t.Fatalf("RenderIndex() error = %v", err)
	}
	if !strings.Contains(buf.String(), "<!DOCTYPE html>") {
		t.Fatalf("RenderIndex() should produce a full HTML page")
	}
}

func TestRenderChannel(t *testing.T) {
	v := newTestViewer(renderer.ModeLive)
	var buf bytes.Buffer
	if err := v.RenderChannel(context.Background(), "C1", &buf); err != nil {
		t.Fatalf("RenderChannel() error = %v", err)
	}
	body := buf.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("RenderChannel() should produce a full HTML page")
	}
}

func TestRenderThread(t *testing.T) {
	v := newTestViewer(renderer.ModeLive)
	var buf bytes.Buffer
	if err := v.RenderThread(context.Background(), "C1", "1234567890.000100", &buf); err != nil {
		t.Fatalf("RenderThread() error = %v", err)
	}
	body := buf.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("RenderThread() should produce a full HTML page")
	}
}

func TestRenderUser(t *testing.T) {
	v := newTestViewer(renderer.ModeLive)
	var buf bytes.Buffer
	if err := v.RenderUser(context.Background(), "U1", &buf); err != nil {
		t.Fatalf("RenderUser() error = %v", err)
	}
	body := buf.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("RenderUser() should produce a full HTML page")
	}
	if !strings.Contains(body, "Profile") {
		t.Fatalf("RenderUser() response should include profile panel")
	}
}

func TestRenderCanvas(t *testing.T) {
	v := newTestViewer(renderer.ModeLive)
	var buf bytes.Buffer
	if err := v.RenderCanvas(context.Background(), "C1", &buf); err != nil {
		t.Fatalf("RenderCanvas() error = %v", err)
	}
	body := buf.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("RenderCanvas() should produce a full HTML page")
	}
}

// TestRenderChannel_StaticMode verifies that static-mode output contains no hx-* attributes.
func TestRenderChannel_StaticMode(t *testing.T) {
	v := newTestViewer(renderer.ModeStatic)
	var buf bytes.Buffer
	if err := v.RenderChannel(context.Background(), "C1", &buf); err != nil {
		t.Fatalf("RenderChannel() static error = %v", err)
	}
	body := buf.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Fatalf("RenderChannel() static should produce a full HTML page")
	}
	if strings.Contains(body, `hx-get`) || strings.Contains(body, `hx-boost`) || strings.Contains(body, `hx-push-url`) {
		t.Fatalf("RenderChannel() static mode should not contain HTMX attributes, got: %q", body)
	}
}
