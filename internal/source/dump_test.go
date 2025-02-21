package source

import (
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/testutil"
	"github.com/rusq/slackdump/v3/types"
)

func Test_convertMessages(t *testing.T) {
	type args struct {
		cm []types.Message
	}
	tests := []struct {
		name string
		args args
		want []testutil.IterVal[slack.Message, error]
	}{
		{
			name: "empty",
			args: args{cm: []types.Message{}},
			want: []testutil.IterVal[slack.Message, error]{},
		},
		{
			name: "one",
			args: args{cm: []types.Message{
				{Message: slack.Message{Msg: slack.Msg{Text: "one"}}},
			}},
			want: []testutil.IterVal[slack.Message, error]{
				{T: slack.Message{Msg: slack.Msg{Text: "one"}}, U: nil},
			},
		},
		{
			name: "two",
			args: args{cm: []types.Message{
				{Message: slack.Message{Msg: slack.Msg{Text: "one"}}},
				{Message: slack.Message{Msg: slack.Msg{Text: "two"}}},
			}},
			want: []testutil.IterVal[slack.Message, error]{
				{T: slack.Message{Msg: slack.Msg{Text: "one"}}, U: nil},
				{T: slack.Message{Msg: slack.Msg{Text: "two"}}, U: nil},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := convertMessages(tt.args.cm)
			var i int
			for m, err := range it {
				assert.Equal(t, tt.want[i].T, m)
				assert.Equal(t, tt.want[i].U, err)
				i++
			}
		})
	}
}
