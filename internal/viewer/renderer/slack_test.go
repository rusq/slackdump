package renderer

import (
	"context"
	"html/template"
	"reflect"
	"strings"
	"testing"

	"github.com/rusq/slack"
)

func TestSlack_Render(t *testing.T) {
	nestedLists := load(t, fxtrMsgNestedLists)
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
			&Slack{},
			args{
				m: load(t, fxtrRtseText),
			},
			template.HTML("New message"),
		},
		{
			"nested lists",
			&Slack{},
			args{
				m: nestedLists,
			},
			template.HTML(`Enumerated:<br><ol><li>First (1)</li><li>Second(2)</li></ol><ol><ol><li>Nested (2.a)</li><li>Nested (2.b)</li></ol></ol><ul><ul><ul><li>Nexted bullet point</li></ul></ul></ul><ul><ul><ul><ul><li>Another nested bullet</li></ul></ul></ul></ul><ol><ol><ol><ol><ol><li>Nested enumeration</li></ol></ol></ol></ol></ol>`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &Slack{}
			if gotV := sm.Render(context.Background(), tt.args.m); !reflect.DeepEqual(gotV, tt.wantV) {
				t.Errorf("Slack.Render() = %v, want %v", gotV, tt.wantV)
			}
		})
	}
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
