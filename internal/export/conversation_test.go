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

// testConversation unmashals and returns the test conversation from fixtures.
func testConversation() slackdump.Conversation {
	var ret slackdump.Conversation
	if err := json.Unmarshal([]byte(fixtures.TestConversationJSON), &ret); err != nil {
		panic(err)
	}
	return ret
}

func TestConversation_ByDate(t *testing.T) {
	// TODO test users

	exp := Export{}
	c := testConversation()
	convDt, err := exp.ByDate()
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
