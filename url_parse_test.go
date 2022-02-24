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
		want    *urlInfo
		wantErr bool
	}{
		{
			name:    "channel",
			args:    args{"https://ora600.slack.com/archives/CHM82GF99"},
			want:    &urlInfo{Channel: "CHM82GF99"},
			wantErr: false,
		},
		{
			name:    "thread",
			args:    args{"https://ora600.slack.com/archives/CHM82GF99/p1577694990000400"},
			want:    &urlInfo{Channel: "CHM82GF99", ThreadTS: "1577694990.000400"},
			wantErr: false,
		},
		{
			name:    "thread with extra data in the URL",
			args:    args{"https://ora600.slack.com/archives/CHM82GF99/p1577694990000400/xxxx"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "malformed thread id",
			args:    args{"https://ora600.slack.com/archives/CHM82GF99/1577694990000400"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid thread id",
			args:    args{"https://ora600.slack.com/archives/CHM82GF99/p15776949900x0400"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "DM",
			args:    args{"https://ora600.slack.com/archives/DL98HT3QA"},
			want:    &urlInfo{Channel: "DL98HT3QA"},
			wantErr: false,
		},
		{
			name:    "Invalid url",
			args:    args{"https://example.com"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid slack url",
			args:    args{"https://app.slack.com/client/THX2HTY8U/CHM82GF99/thread/CHM82GF99-1577694990.000400"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "malformed",
			args:    args{"https://ora600.slack.com/archives/"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "not a url",
			args:    args{"C123454321"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty",
			args:    args{""},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "binary junk",
			args:    args{"\x02"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "thread",
			args:    args{"https://xxxxxx.slack.com/archives/CHANNEL/p1645551829244659"},
			want:    &urlInfo{Channel: "CHANNEL", ThreadTS: "1645551829.244659"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseURL(tt.args.slackURL)
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

func TestURLInfo_IsThread(t *testing.T) {
	type fields struct {
		Channel  string
		ThreadTS string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"yes", fields{Channel: "x", ThreadTS: "x"}, true},
		{"no", fields{Channel: "x", ThreadTS: ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := urlInfo{
				Channel:  tt.fields.Channel,
				ThreadTS: tt.fields.ThreadTS,
			}
			if got := u.IsThread(); got != tt.want {
				t.Errorf("URLInfo.IsThread() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestURLInfo_IsValid(t *testing.T) {
	type fields struct {
		Channel  string
		ThreadTS string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"channel", fields{Channel: "x"}, true},
		{"thread", fields{Channel: "x", ThreadTS: "y"}, true},
		{"invalid", fields{ThreadTS: "y"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := urlInfo{
				Channel:  tt.fields.Channel,
				ThreadTS: tt.fields.ThreadTS,
			}
			if got := u.IsValid(); got != tt.want {
				t.Errorf("URLInfo.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
