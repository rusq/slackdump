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

func TestAvatarProc_removeDoubleDots(t *testing.T) {
	type fields struct {
		dl       Downloader
		filepath func(u *slack.User) string
	}
	type args struct {
		uri string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "good URI",
			args: args{"https://example.com/facepalm.jpg"},
			want: "https://example.com/facepalm.jpg",
		},
		{
			name: "double full stop in the URI",
			args: args{"https://example.com/facepalm..jpg"},
			want: "https://example.com/facepalm.jpg",
		},
		{
			name: "zero length",
			args: args{""},
			want: "",
		},
		{
			name: "just the extension",
			args: args{".png"},
			want: ".png",
		},
		{
			name: "extension and double full stop",
			args: args{"..tiff"},
			want: ".tiff",
		},
		{
			name: "no extension",
			args: args{"https://example.com/buriburi"},
			want: "https://example.com/buriburi",
		},
		{
			name: "non-ascii",
			args: args{"ぶりぶり..jpg"},
			want: "ぶりぶり.jpg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := AvatarProc{
				dl:       tt.fields.dl,
				filepath: tt.fields.filepath,
			}
			if got := a.removeDoubleDots(tt.args.uri); got != tt.want {
				t.Errorf("AvatarProc.removeDoubleDots() = %v, want %v", got, tt.want)
			}
		})
	}
}
