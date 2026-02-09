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
	"html/template"
	"log/slog"
	"net/http"
	"testing"

	"github.com/rusq/slackdump/v4/source"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/fixtures"
	st "github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/internal/viewer/renderer"
)

func TestViewer_username(t *testing.T) {
	type fields struct {
		ch   channels
		um   st.UserIndex
		rtr  source.Sourcer
		tmpl *template.Template
		srv  *http.Server
		lg   *slog.Logger
		r    renderer.Renderer
	}
	type args struct {
		m slack.Message
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			"bot message",
			fields{
				um: st.UserIndex{},
				lg: slog.Default(),
			},
			args{
				m: fixtures.Load[slack.Message](fixtures.AppMessageJSON),
			},
			"WakaTime (app)",
		},
		{
			"bot message with empty username",
			fields{
				um: st.UserIndex{},
				lg: slog.Default(),
			},
			args{
				m: fixtures.Load[slack.Message](fixtures.BotMessageJSON),
			},
			"Unknown user via BUEBX9AUR (bot)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Viewer{
				ch:   tt.fields.ch,
				um:   tt.fields.um,
				src:  tt.fields.rtr,
				tmpl: tt.fields.tmpl,
				srv:  tt.fields.srv,
				lg:   tt.fields.lg,
				r:    tt.fields.r,
			}
			if got := v.username(tt.args.m); got != tt.want {
				t.Errorf("Viewer.username() = %v, want %v", got, tt.want)
			}
		})
	}
}
