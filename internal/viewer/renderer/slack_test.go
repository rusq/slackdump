package renderer

import (
	"html/template"
	"reflect"
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
			"nested lists",
			&Slack{},
			args{
				m: nestedLists,
			},
			template.HTML(`numerated:<br><ol><li>First (1)</li><li>Second(2)</li></ol><ol><ol><li>Nested (2.a)</li><li>Nested (2.b)</li></ol></ol><ul><ul><ul><li>Nexted bullet point</li></ul></ul></ul><ul><ul><ul><ul><li>Another nested bullet</li></ul></ul></ul></ul><ol><ol><ol><ol><ol><li>Nested enumeration</li></ol></ol></ol></ol></ol>`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &Slack{}
			if gotV := sm.Render(tt.args.m); !reflect.DeepEqual(gotV, tt.wantV) {
				t.Errorf("Slack.Render() = %v, want %v", gotV, tt.wantV)
			}
		})
	}
}
