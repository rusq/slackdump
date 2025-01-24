package renderer

import (
	"bytes"
	"compress/gzip"
	"context"
	"html/template"
	"io"
	"strings"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/viewer/renderer/functions"
)

var tmpl = template.Must(template.New("blocks").Funcs(functions.FuncMap).ParseFS(templates, "templates/*.html"))

func TestSlack_Render(t *testing.T) {
	nestedLists := loadmsg(t, fxtrMsgNestedLists)
	type args struct {
		m *slack.Message
	}
	tests := []struct {
		name  string
		sm    *Slack
		args  args
		wantV template.HTML
	}{
		{
			"simple message",
			&Slack{
				tmpl: tmpl,
			},
			args{
				m: loadmsg(t, fxtrRtseText),
			},
			template.HTML("New message"),
		},
		{
			"nested lists",
			&Slack{
				tmpl: tmpl,
			},
			args{
				m: nestedLists,
			},
			template.HTML(`Enumerated:<br><ol><li>First (1)</li><li>Second(2)</li></ol><ol><ol><li>Nested (2.a)</li><li>Nested (2.b)</li></ol></ol><ul><ul><ul><li>Nexted bullet point</li></ul></ul></ul><ul><ul><ul><ul><li>Another nested bullet</li></ul></ul></ul></ul><ol><ol><ol><ol><ol><li>Nested enumeration</li></ol></ol></ol></ol></ol>`),
		},
		{
			"template panic",
			&Slack{
				tmpl: tmpl,
			},
			args{
				m: loadmsg(t, fxtrRtseText),
			},
			template.HTML("New message"),
		},
		{
			"started a meeting",
			&Slack{
				tmpl: tmpl,
			},
			args{
				m: loadmsg(t, fxtrStartedAMeeting),
			},
			template.HTML(`<div class="slack-call">(Call)</div><pre class="slack-section-text">Meeting passcode: yyyyy</pre>`),
		},
		{
			"polly message",
			&Slack{
				tmpl: tmpl,
			},
			args{
				m: loadmsg(t, fxtrPolly),
			},
			template.HTML(strings.TrimSpace(ungzip(t, fxtrPollyHTML))),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &Slack{}
			gotV := sm.Render(context.Background(), tt.args.m)
			assert.Equal(t, tt.wantV, gotV)
		})
	}
}

func ungzip(t *testing.T, b []byte) string {
	t.Helper()
	gr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	var buf strings.Builder
	if _, err := io.Copy(&buf, gr); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestSlack_renderAttachment(t *testing.T) {
	type fields struct {
		tmpl *template.Template
		uu   map[string]slack.User
		cc   map[string]slack.Channel
	}
	type args struct {
		ctx   context.Context
		buf   *strings.Builder
		msgTS string
		a     slack.Attachment
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slack{
				tmpl: tt.fields.tmpl,
				uu:   tt.fields.uu,
				cc:   tt.fields.cc,
			}
			s.renderAttachment(tt.args.ctx, tt.args.buf, tt.args.msgTS, tt.args.a)
		})
	}
}
