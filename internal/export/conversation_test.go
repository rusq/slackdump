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
	// TODO
	var exp Export
	conversations := fixtures.Load[slackdump.Conversation](fixtures.TestConversationJSON)
	convDt, err := exp.byDate(&conversations, fixtures.Load[slackdump.Users](fixtures.UsersJSON))
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

func Test_messagesByDate_validate(t *testing.T) {
	tests := []struct {
		name    string
		mbd     messagesByDate
		wantErr bool
	}{
		{"valid",
			messagesByDate{
				"2019-09-16": []ExportMessage{},
				"2020-12-31": []ExportMessage{},
			},
			false,
		},
		{"empty key",
			messagesByDate{
				"":           []ExportMessage{},
				"2020-12-31": []ExportMessage{},
			},
			true,
		},
		{"invalid key",
			messagesByDate{
				"2019-09-16": []ExportMessage{},
				"2020-31-12": []ExportMessage{}, //swapped month and date
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.mbd.validate(); (err != nil) != tt.wantErr {
				t.Errorf("messagesByDate.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
