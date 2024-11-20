// Package structures provides functions to parse Slack data types.
package structures

import (
	"testing"

	"github.com/rusq/slackdump/v3/internal/fixtures"
)

func TestValidateToken(t *testing.T) {
	type args struct {
		token string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "app token",
			args:    args{token: fixtures.TestAppToken},
			wantErr: false,
		},
		{
			name:    "bot token",
			args:    args{token: fixtures.TestBotToken},
			wantErr: false,
		},
		{
			name:    "client token",
			args:    args{token: fixtures.TestClientToken},
			wantErr: false,
		},
		{
			name:    "export token",
			args:    args{token: fixtures.TestExportToken},
			wantErr: false,
		},
		{
			name:    "legacy token",
			args:    args{token: fixtures.TestPersonalToken},
			wantErr: false,
		},
		{
			name:    "invalid prefix",
			args:    args{token: "xoxz-123456789012-123456789012-123456789012-12345678901234567890123456789012"},
			wantErr: true,
		},
		{
			name:    "short token",
			args:    args{token: "xoxa-123456789012-123456789012-123456789012-1234567890123456789012345678901"},
			wantErr: true,
		},
		{
			name:    "long token",
			args:    args{token: "xoxa-123456789012-123456789012-123456789012-123456789012345678901234567890123"},
			wantErr: true,
		},
		{
			name:    "non-numeric sections",
			args:    args{token: "xoxa-123456789012-abcdefg-123456789012-12345678901234567890123456789012"},
			wantErr: true,
		},
		{
			name:    "non-alphanumeric suffix",
			args:    args{token: "xoxa-123456789012-123456789012-123456789012-1234567890123456789012345678901!"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateToken(tt.args.token); (err != nil) != tt.wantErr {
				t.Errorf("validateToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractWorkspace(t *testing.T) {
	type args struct {
		workspace string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"not a URL", args{"blahblah"}, "blahblah", false},
		{"url slash", args{"https://blahblah.slack.com/"}, "blahblah", false},
		{"url no slash", args{"https://blahblah.slack.com"}, "blahblah", false},
		{"url no schema slash", args{"blahblah.slack.com/"}, "blahblah", false},
		{"url no schema no slash", args{"blahblah.slack.com"}, "blahblah", false},
		{"not a slack domain", args{"blahblah.example.com"}, "", true},
		{"enterprise domain", args{"https://grid-lightspeedhq.enterprise.slack.com/"}, "grid-lightspeedhq", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractWorkspace(tt.args.workspace)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractWorkspace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractWorkspace() got = %v, want %v", got, tt.want)
			}
		})
	}
}
