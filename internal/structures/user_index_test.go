package structures

import (
	"testing"

	"github.com/rusq/slackdump/v2/internal/fixtures"
)

func TestUserIndex_IsDeleted(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name string
		idx  UserIndex
		args args
		want bool
	}{
		{
			name: "deleted",
			idx:  NewUserIndex(fixtures.TestUsers),
			args: args{"DELD"},
			want: true,
		},
		{
			name: "not deleted",
			idx:  NewUserIndex(fixtures.TestUsers),
			args: args{"LOL1"},
			want: false,
		},
		{
			name: "not present",
			idx:  NewUserIndex(fixtures.TestUsers),
			args: args{"XXX"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.idx.IsDeleted(tt.args.id); got != tt.want {
				t.Errorf("UserIndex.IsDeleted() = %v, want %v", got, tt.want)
			}
		})
	}

}
