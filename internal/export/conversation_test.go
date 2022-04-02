package export

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/rusq/slackdump"
	"github.com/rusq/slackdump/internal/fixtures"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestConversation_ByDate(t *testing.T) {
	var exp Export
	c := fixtures.Load[slackdump.Conversation](fixtures.TestConversationJSON)
	convDt, err := exp.byDate(&c, fixtures.Load[slackdump.Users](fixtures.UsersJSON))
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(convDt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("x.json", data, 0644); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, map[string][]slack.Conversation{}, convDt)
}
