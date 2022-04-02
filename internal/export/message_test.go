package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_makeUniq(t *testing.T) {
	expMsg := ExportMessage{
		ReplyUsers: []string{"A", "A", "C", "B"},
	}
	makeUniq(&expMsg.ReplyUsers)
	assert.Equal(t, []string{"A", "B", "C"}, expMsg.ReplyUsers)
}
