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
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGetExpectedChecksum(t *testing.T) {
	tests := []struct {
		name           string
		checksumData   string
		assetName      string
		wantHash       string
		wantErr        bool
		includeAsset   bool
		checksumStatus int
	}{
		{
			name: "valid checksum found",
			checksumData: `7b64d5722e18e5802335e87b06617702b957f6862549f5054a768023e74dd43b  slackdump_Linux_x86_64.tar.gz
298254e5604e5ce218ea026d25d52a4358f2542ed04d52671d4e806931ee2f49  slackdump_Openbsd_arm64.tar.gz
8b76c75db5dda4d16d018c1742cbaf6f24811bcc5a5db74d0a40abfd04887888  slackdump_Windows_arm64.zip`,
			assetName:      "slackdump_Linux_x86_64.tar.gz",
			wantHash:       "7b64d5722e18e5802335e87b06617702b957f6862549f5054a768023e74dd43b",
			wantErr:        false,
			includeAsset:   true,
			checksumStatus: http.StatusOK,
		},
		{
			name: "checksum not found for asset",
			checksumData: `7b64d5722e18e5802335e87b06617702b957f6862549f5054a768023e74dd43b  slackdump_Linux_x86_64.tar.gz
298254e5604e5ce218ea026d25d52a4358f2542ed04d52671d4e806931ee2f49  slackdump_Openbsd_arm64.tar.gz`,
			assetName:      "slackdump_nonexistent.tar.gz",
			wantErr:        true,
			includeAsset:   true,
			checksumStatus: http.StatusOK,
		},
		{
			name:           "checksums.txt not in release",
			assetName:      "slackdump_Linux_x86_64.tar.gz",
			wantErr:        true,
			includeAsset:   false,
			checksumStatus: http.StatusOK,
		},
		{
			name:           "checksums.txt download fails",
			checksumData:   "some data",
			assetName:      "slackdump_Linux_x86_64.tar.gz",
			wantErr:        true,
			includeAsset:   true,
			checksumStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server to serve checksums.txt
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.checksumStatus)
				if tt.checksumStatus == http.StatusOK {
					if _, err := w.Write([]byte(tt.checksumData)); err != nil {
						t.Errorf("Failed to write test data: %v", err)
					}
				}
			}))
			defer server.Close()

			// Create a release with checksums.txt asset
			var assets []github.Asset
			if tt.includeAsset {
				assets = append(assets, github.Asset{
					Name:               "checksums.txt",
					BrowserDownloadURL: server.URL + "/checksums.txt",
				})
			}

			release := &github.Release{
				Assets: assets,
			}

			hash, err := getExpectedChecksum(context.Background(), release, tt.assetName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getExpectedChecksum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hash != tt.wantHash {
				t.Errorf("getExpectedChecksum() hash = %v, want %v", hash, tt.wantHash)
			}
		})
	}
}

func TestDownloadAssetWithChecksum(t *testing.T) {
	validChecksum := "5feceb66ffc86f38d952786c6d696c79c2dbc239dd4e91b46729d73a27fb57e9" // SHA256 of "0"
	invalidChecksum := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	tests := []struct {
		name            string
		assetData       string
		checksumData    string
		includeChecksum bool
		wantErr         bool
		expectMismatch  bool
	}{
		{
			name:      "valid download with matching checksum",
			assetData: "0",
			checksumData: validChecksum + "  test_asset.tar.gz\n" +
				"other_hash  other_file.tar.gz",
			includeChecksum: true,
			wantErr:         false,
		},
		{
			name:      "valid download with mismatched checksum",
			assetData: "0",
			checksumData: invalidChecksum + "  test_asset.tar.gz\n" +
				"other_hash  other_file.tar.gz",
			includeChecksum: true,
			wantErr:         true,
			expectMismatch:  true,
		},
		{
			name:            "valid download without checksum file",
			assetData:       "0",
			includeChecksum: false,
			wantErr:         false, // Should succeed with warning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test servers
			assetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(tt.assetData)); err != nil {
					t.Errorf("Failed to write test asset data: %v", err)
				}
			}))
			defer assetServer.Close()

			checksumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(tt.checksumData)); err != nil {
					t.Errorf("Failed to write test checksum data: %v", err)
				}
			}))
			defer checksumServer.Close()

			// Create release and asset
			var assets []github.Asset
			assets = append(assets, github.Asset{
				Name:               "test_asset.tar.gz",
				BrowserDownloadURL: assetServer.URL + "/test_asset.tar.gz",
			})
			if tt.includeChecksum {
				assets = append(assets, github.Asset{
					Name:               "checksums.txt",
					BrowserDownloadURL: checksumServer.URL + "/checksums.txt",
				})
			}

			release := &github.Release{
				Assets: assets,
			}

			downloadPath, err := downloadAsset(context.Background(), release, &assets[0])
			if (err != nil) != tt.wantErr {
				t.Errorf("downloadAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expectMismatch && err != nil {
				if !strings.Contains(err.Error(), "checksum mismatch") {
					t.Errorf("Expected checksum mismatch error, got: %v", err)
				}
			}

			// Clean up downloaded file if it exists
			if downloadPath != "" {
				// File will be cleaned up by defer in downloadAsset
				t.Logf("Downloaded to: %s", downloadPath)
			}
		})
	}
}
