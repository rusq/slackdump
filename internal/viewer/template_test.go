package viewer

import (
	"html/template"
	"net/http"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	st "github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/internal/viewer/renderer"
	"github.com/rusq/slackdump/v3/logger"
)

func TestViewer_username(t *testing.T) {
	type fields struct {
		ch   channels
		um   st.UserIndex
		rtr  Sourcer
		tmpl *template.Template
		srv  *http.Server
		lg   logger.Interface
		r    renderer.Renderer
	}
	type args struct {
		m *slack.Message
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
			},
			args{
				m: fixtures.Load[*slack.Message](fixtures.AppMessageJSON),
			},
			"WakaTime (app)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Viewer{
				ch:   tt.fields.ch,
				um:   tt.fields.um,
				rtr:  tt.fields.rtr,
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
