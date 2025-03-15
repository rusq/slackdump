package fileproc

import (
	"path/filepath"
	"testing"

	"github.com/rusq/slack"
)

func Test_avatarPath(t *testing.T) {
	type args struct {
		u *slack.User
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "name with display name",
			args: args{
				u: &slack.User{
					ID: "U12345678",
					Profile: slack.UserProfile{
						ImageOriginal:         "https://example/image.jpg",
						DisplayNameNormalized: "displayname",
					},
				},
			},
			want: filepath.Join("__avatars", "U12345678", "image.jpg"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AvatarPath(tt.args.u); got != tt.want {
				t.Errorf("avatarPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
