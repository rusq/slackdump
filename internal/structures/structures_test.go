// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
// Package structures provides functions to parse Slack data types.
package structures

import (
	"reflect"
	"testing"
	"time"

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
			name:    "i562, oauth token with 32 chars in the last section",
			args:    args{token: fixtures.TestOauthToken},
			wantErr: false,
		},
		{
			name:    "invalid prefix",
			args:    args{token: "xoxz-123456789012-123456789012-123456789012-12345678901234567890123456789012"},
			wantErr: true,
		},
		{
			name:    "short token",
			args:    args{token: "xoxc-123456789012-123456789012-123456789012-1234567890123456789012345678901"},
			wantErr: true,
		},
		{
			name:    "long token",
			args:    args{token: "xoxc-123456789012-123456789012-123456789012-123456789012345678901234567890123123456789012345678901234567890123"},
			wantErr: true,
		},
		{
			name:    "non-numeric sections",
			args:    args{token: "xoxc-123456789012-abcdefg-123456789012-12345678901234567890123456789012"},
			wantErr: true,
		},
		{
			name:    "non-alphanumeric suffix",
			args:    args{token: "xoxc-123456789012-123456789012-123456789012-1234567890123456789012345678901!"},
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
		{"enterprise domain", args{"https://acme-co.enterprise.slack.com/"}, "acme-co", false},
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

func TestNVLTime(t *testing.T) {
	type args struct {
		t   time.Time
		def time.Time
	}
	tests := []struct {
		name string
		args args
		want time.Time
	}{
		{
			"t is zero",
			args{
				time.Time{},
				time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"t is not zero",
			args{
				time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NVLTime(tt.args.t, tt.args.def); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("nvlTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
