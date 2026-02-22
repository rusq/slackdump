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

package updater

import (
	"testing"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/diag/updater/github"
)

func TestFindAsset(t *testing.T) {
	release := &github.Release{
		Assets: []github.Asset{
			{Name: "slackdump_v2.3.4_Darwin_x86_64.zip", BrowserDownloadURL: "https://example.com/darwin-amd64.zip"},
			{Name: "slackdump_v2.3.4_Darwin_arm64.zip", BrowserDownloadURL: "https://example.com/darwin-arm64.zip"},
			{Name: "slackdump_v2.3.4_Linux_x86_64.zip", BrowserDownloadURL: "https://example.com/linux-amd64.zip"},
			{Name: "slackdump_v2.3.4_Linux_arm64.zip", BrowserDownloadURL: "https://example.com/linux-arm64.zip"},
			{Name: "slackdump_v2.3.4_Windows_x86_64.zip", BrowserDownloadURL: "https://example.com/windows-amd64.zip"},
		},
	}

	tests := []struct {
		name    string
		osName  string
		arch    string
		wantURL string
		wantErr bool
	}{
		{
			name:    "darwin amd64",
			osName:  "darwin",
			arch:    "amd64",
			wantURL: "https://example.com/darwin-amd64.zip",
			wantErr: false,
		},
		{
			name:    "darwin arm64",
			osName:  "darwin",
			arch:    "arm64",
			wantURL: "https://example.com/darwin-arm64.zip",
			wantErr: false,
		},
		{
			name:    "linux amd64",
			osName:  "linux",
			arch:    "amd64",
			wantURL: "https://example.com/linux-amd64.zip",
			wantErr: false,
		},
		{
			name:    "windows amd64",
			osName:  "windows",
			arch:    "amd64",
			wantURL: "https://example.com/windows-amd64.zip",
			wantErr: false,
		},
		{
			name:    "unsupported os",
			osName:  "freebsd",
			arch:    "amd64",
			wantErr: true,
		},
		{
			name:    "unsupported arch",
			osName:  "linux",
			arch:    "mips",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := findAsset(release, tt.osName, tt.arch)
			if (err != nil) != tt.wantErr {
				t.Errorf("findAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if asset == nil {
					t.Error("findAsset() returned nil asset")
					return
				}
				if asset.BrowserDownloadURL != tt.wantURL {
					t.Errorf("findAsset() URL = %v, want %v", asset.BrowserDownloadURL, tt.wantURL)
				}
			}
		})
	}
}
