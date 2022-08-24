package files

import (
	"testing"
)

func Test_addToken(t *testing.T) {
	type args struct {
		uri   string
		token string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"ok",
			args{"https://slack.com/files/BLAHBLAH/x.jpg", "xoxe-xxxxx"},
			"https://slack.com/files/BLAHBLAH/x.jpg?t=xoxe-xxxxx",
			false,
		},
		{
			"replace existing",
			args{"https://slack.com/files/BLAHBLAH/x.jpg?t=xoxe-yyyyy", "xoxe-xxxxx"},
			"https://slack.com/files/BLAHBLAH/x.jpg?t=xoxe-xxxxx",
			false,
		},
		{
			"preseves other parameters",
			args{"https://slack.com/files/BLAHBLAH/x.jpg?t=xoxe-yyyyy&q=bbbb", "xoxe-xxxxx"},
			"https://slack.com/files/BLAHBLAH/x.jpg?q=bbbb&t=xoxe-xxxxx",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := addToken(tt.args.uri, tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("addToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("addToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
