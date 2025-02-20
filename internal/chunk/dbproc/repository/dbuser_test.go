package repository

import (
	"reflect"
	"testing"

	"github.com/rusq/slack"
)

var user1 = &slack.User{
	ID:       "U123",
	TeamID:   "T777",
	Name:     "bob",
	Deleted:  false,
	Color:    "#ff0000", // roses are red, violets are red, everything is red, hungry, poor and sad
	RealName: "Dominic Decocco",
	TZ:       "Pacific/Auckland",
	TZLabel:  "NZDT",
	TZOffset: 46800,
	Profile: slack.UserProfile{
		FirstName:             "Dominic",
		LastName:              "Decocco",
		RealName:              "",
		RealNameNormalized:    "",
		DisplayName:           "dom",
		DisplayNameNormalized: "",
		Team:                  "T777",
	},
	Has2FA:        true,
	TwoFactorType: new(string),
	Updated:       1725318212,
	Enterprise:    slack.EnterpriseUser{},
}

func TestNewDBUser(t *testing.T) {
	type args struct {
		chunkID int64
		n       int
		u       *slack.User
	}
	tests := []struct {
		name    string
		args    args
		want    *DBUser
		wantErr bool
	}{
		{
			name: "creates a new DBUser",
			args: args{chunkID: 42, n: 50, u: user1},
			want: &DBUser{
				ID:       "U123",
				ChunkID:  42,
				Username: "bob",
				Index:    50,
				Data:     must(marshal(user1)),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDBUser(tt.args.chunkID, tt.args.n, tt.args.u)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDBUser() = %v, want %v", got, tt.want)
			}
		})
	}
}
