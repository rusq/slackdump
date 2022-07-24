package export

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

func Test_makeUniq(t *testing.T) {
	expMsg := ExportMessage{
		ReplyUsers: []string{"A", "A", "C", "B"},
	}
	makeUniq(&expMsg.ReplyUsers)
	assert.Equal(t, []string{"A", "B", "C"}, expMsg.ReplyUsers)
}

func Test_newExportMessage(t *testing.T) {
	type args struct {
		msg   *types.Message
		users structures.UserIndex
	}
	tests := []struct {
		name string
		args args
		want *ExportMessage
	}{
		{
			"threaded message fields are populated correctly",
			args{
				msg:   fixtures.Load[*types.Message](fixtures.ThreadMessage1JSON),
				users: fixtures.Load[types.Users](fixtures.UsersJSON).IndexByID(),
			},
			fixtures.Load[*ExportMessage](fixtures.ThreadedExportedMessage1JSON),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newExportMessage(tt.args.msg, tt.args.users)
			assert.Equal(t, tt.want, got)
		})
	}
}
