package export

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestConversation_ByDate(t *testing.T) {
	// TODO
	var exp Export

	conversations := fixtures.Load[slackdump.Conversation](fixtures.TestConversationJSON)
	users := fixtures.Load[slackdump.Users](fixtures.UsersJSON)

	convDt, err := exp.byDate(&conversations, users)
	if err != nil {
		t.Fatal(err)
	}

	// uncomment to write the json for fixtures
	// require.NoError(t, writeOutput("convDt", convDt))

	want := fixtures.Load[map[string][]ExportMessage](fixtures.TestConversationExportJSON)
	assert.Equal(t, want, convDt)
}

func writeOutput(name string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(name+".json", data, 0644); err != nil {
		return err
	}
	return nil
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
