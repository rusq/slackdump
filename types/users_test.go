package types

import (
	"bytes"
	"testing"

	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/stretchr/testify/assert"
)

var testUsers = Users(fixtures.TestUsers)

func TestUsers_ToText(t *testing.T) {
	tests := []struct {
		name    string
		us      Users
		wantW   string
		wantErr bool
	}{
		{
			"test user list",
			testUsers,
			"Name          ID    Bot?  Deleted?  Restricted?\n                                    \nka            DELD        deleted   \nmotherfucker  LOL4  bot             \nyay           LOL3                  restricted\nyippi         LOL1                  \n",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := tt.us.ToText(w, nil); (err != nil) != tt.wantErr {
				t.Errorf("Users.ToText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.wantW, w.String())
		})
	}
}
