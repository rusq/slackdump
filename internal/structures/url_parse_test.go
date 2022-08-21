package structures

import (
	"reflect"
	"testing"
)

const (
	sampleChannelURL     = "https://ora600.slack.com/archives/CHM82GF99"
	sampleThreadURL      = "https://ora600.slack.com/archives/CHM82GF99/p1577694990000400"
	sampleThreadWDashURL = "https://ora-600.slack.com/archives/CHM82GF99/p1577694990000400"
	sampleDMURL          = "https://ora600.slack.com/archives/DL98HT3QA"

	sampleChannelID = "CHM82GF99"
)

func TestParseURL(t *testing.T) {
	type args struct {
		slackURL string
	}
	tests := []struct {
		name    string
		args    args
		want    *SlackLink
		wantErr bool
	}{
		{
			name:    "channel",
			args:    args{sampleChannelURL},
			want:    &SlackLink{Channel: "CHM82GF99"},
			wantErr: false,
		},
		{
			name:    "thread",
			args:    args{sampleThreadURL},
			want:    &SlackLink{Channel: "CHM82GF99", ThreadTS: "1577694990.000400"},
			wantErr: false,
		},
		{
			name:    "thread",
			args:    args{sampleThreadWDashURL},
			want:    &SlackLink{Channel: "CHM82GF99", ThreadTS: "1577694990.000400"},
			wantErr: false,
		},
		{
			name:    "thread with extra data in the URL",
			args:    args{sampleThreadURL + "/xxxx"},
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
			args:    args{sampleDMURL},
			want:    &SlackLink{Channel: "DL98HT3QA"},
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
			want:    &SlackLink{Channel: "CHANNEL", ThreadTS: "1645551829.244659"},
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
			u := SlackLink{
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
			u := SlackLink{
				Channel:  tt.fields.Channel,
				ThreadTS: tt.fields.ThreadTS,
			}
			if got := u.IsValid(); got != tt.want {
				t.Errorf("URLInfo.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidSlackURL(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"channel url",
			args{sampleChannelURL},
			true,
		},
		{
			"thread url",
			args{sampleThreadURL},
			true,
		},
		{
			"invalid thread URL",
			args{"https://ora600.slack.com/archives/CHM82GF99/p15776949900x0400"},
			false,
		},
		{
			"is no url",
			args{"C43012851"},
			false,
		},
		{
			"empty",
			args{""},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSlackURL(tt.args.s); got != tt.want {
				t.Errorf("IsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLink(t *testing.T) {
	type args struct {
		link string
	}
	tests := []struct {
		name    string
		args    args
		want    SlackLink
		wantErr bool
	}{
		{
			"channel ID",
			args{"C4810"},
			SlackLink{Channel: "C4810"},
			false,
		},
		{
			"channel ID and thread TS",
			args{"C4810" + linkSep + "1577694990.000400"},
			SlackLink{Channel: "C4810", ThreadTS: "1577694990.000400"},
			false,
		},
		{
			"url",
			args{sampleChannelURL},
			SlackLink{Channel: sampleChannelID},
			false,
		},
		{
			"thread URL",
			args{sampleThreadURL},
			SlackLink{Channel: sampleChannelID, ThreadTS: "1577694990.000400"},
			false,
		},
		{
			"invalid URL",
			args{"https://example.com"},
			SlackLink{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLink(tt.args.link)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLink() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseLink() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
