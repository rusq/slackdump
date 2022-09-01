package structures

import (
	"reflect"
	"testing"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/fixtures"
)

var (
	testMpIM          = fixtures.Load[*slack.Channel](fixtures.MpIM)
	testMpIMnoMembers = fixtures.Load[*slack.Channel](fixtures.MpIMNoMembers)
	testMpIMFixed     = fixtures.Load[*slack.Channel](fixtures.MpIMnoMembersFixed)
	testChannel       = fixtures.Load[*slack.Channel](fixtures.TestChannel)
	testMpIMUsers     = fixtures.TestUsers
)

func TestFixMpIMmembers(t *testing.T) {
	type args struct {
		ch    *slack.Channel
		users []slack.User
	}
	tests := []struct {
		name    string
		args    args
		want    *slack.Channel
		wantErr bool
	}{
		{
			"fixed",
			args{ch: testMpIMnoMembers, users: testMpIMUsers},
			testMpIMFixed,
			false,
		},
		{
			"mpim with populated members is untouched",
			args{ch: testMpIM, users: testMpIMUsers},
			testMpIM,
			false,
		},
		{
			"empty users is an error",
			args{ch: testMpIMnoMembers, users: nil},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testCopy slack.Channel = *tt.args.ch
			got, err := FixMpIMmembers(&testCopy, tt.args.users)
			if (err != nil) != tt.wantErr {
				t.Errorf("FixMpIMmembers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FixMpIMmembers() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isMpIM(t *testing.T) {
	type args struct {
		ch *slack.Channel
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"mpim",
			args{testMpIM},
			true,
		},
		{"mpim (group)",
			args{testMpIMnoMembers},
			true,
		},
		{
			"not a mpim (public channel)",
			args{testChannel},
			false,
		},
		{
			"invalid (ismpim, but normalised name is not mpdm",
			args{&slack.Channel{
				GroupConversation: slack.GroupConversation{
					Conversation: slack.Conversation{NameNormalized: "blah", IsMpIM: true},
					Name:         "blah",
				},
			}},
			false,
		},
		{
			"invalid (mpim=false, but normalised name is mpdb",
			args{&slack.Channel{
				GroupConversation: slack.GroupConversation{
					Conversation: slack.Conversation{NameNormalized: "blah", IsMpIM: false},
					Name:         mpimPrefix + "-yay-1",
				},
			}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMpIM(tt.args.ch); got != tt.want {
				t.Errorf("isMpIM() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mpimMemberCount(t *testing.T) {
	type args struct {
		nameNormalized string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"count is correct",
			args{testMpIM.NameNormalized},
			4,
		},
		{
			"count is correct (empty members)",
			args{testMpIMnoMembers.NameNormalized},
			3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mpimMemberCount(tt.args.nameNormalized); got != tt.want {
				t.Errorf("mpimMemberCount(%q) = %v, want %v", tt.args.nameNormalized, got, tt.want)
			}
		})
	}
}

func Test_parseMpIMmembers(t *testing.T) {
	type args struct {
		nn          string
		usernameIDs map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			"mpim with members",
			args{testMpIM.NameNormalized, usernameIDs(testMpIMUsers)},
			[]string{"LOL1", "DELD", "LOL3", "LOL4"},
			false,
		},
		{
			"mpim with no members",
			args{testMpIMnoMembers.NameNormalized, usernameIDs(testMpIMUsers)},
			[]string{"LOL1", "LOL3", "LOL4"},
			false,
		},
		{
			"empty users is an error",
			args{testMpIMnoMembers.NameNormalized, nil},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMpIMmembers(tt.args.nn, tt.args.usernameIDs)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMpIMmembers(%q) error = %v, wantErr %v", tt.args.nn, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseMpIMmembers(%q) got = %v, want %v", tt.args.nn, got, tt.want)
			}
		})
	}
}
