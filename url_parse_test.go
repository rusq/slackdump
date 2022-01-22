package slackdump

import (
	"reflect"
	"testing"
)

func TestParseURL(t *testing.T) {
	type args struct {
		slackURL string
	}
	tests := []struct {
		name    string
		args    args
		want    *URLInfo
		wantErr bool
	}{
		{
			name:    "channel",
			args:    args{"https://ora600.slack.com/archives/CHM82GF99"},
			want:    &URLInfo{Channel: "CHM82GF99"},
			wantErr: false,
		},
		{
			name:    "thread",
			args:    args{"https://ora600.slack.com/archives/CHM82GF99/p1577694990000400"},
			want:    &URLInfo{Channel: "CHM82GF99", Thread: 1577694990000400},
			wantErr: false,
		},
		{
			name:    "DM",
			args:    args{"https://ora600.slack.com/archives/DL98HT3QA"},
			want:    &URLInfo{Channel: "DL98HT3QA"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseURL(tt.args.slackURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseURL() got = %v, want %v", got, tt.want)
			}
		})
	}
}
