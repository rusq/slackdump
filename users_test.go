package slackdump

import (
	"reflect"
	"testing"

	"github.com/slack-go/slack"
)

func TestUsers_MakeUserIDIndex(t *testing.T) {
	users := Users{Users: []slack.User{
		{ID: "USLACKBOT", Name: "slackbot"},
		{ID: "USER2", Name: "User 2"},
	}}
	tests := []struct {
		name string
		us   *Users
		want map[string]*slack.User
	}{
		{"test 1", &users, map[string]*slack.User{
			"USLACKBOT": &users.Users[0],
			"USER2":     &users.Users[1],
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.us.MakeUserIDIndex(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Users.MakeUserIDIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
