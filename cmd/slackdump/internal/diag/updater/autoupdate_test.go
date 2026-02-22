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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/diag/updater/github"
)

func TestFindAsset(t *testing.T) {
	release := &github.Release{
		Assets: []github.Asset{
			{Name: "slackdump_v2.3.4_macOS_x86_64.zip", BrowserDownloadURL: "https://example.com/darwin-amd64.zip"},
			{Name: "slackdump_v2.3.4_macOS_arm64.zip", BrowserDownloadURL: "https://example.com/darwin-arm64.zip"},
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
		{
			name: "filename with spaces",
			checksumData: `7b64d5722e18e5802335e87b06617702b957f6862549f5054a768023e74dd43b  my file with spaces.tar.gz
298254e5604e5ce218ea026d25d52a4358f2542ed04d52671d4e806931ee2f49  another-file.tar.gz`,
			assetName:      "my file with spaces.tar.gz",
			wantHash:       "7b64d5722e18e5802335e87b06617702b957f6862549f5054a768023e74dd43b",
			wantErr:        false,
			includeAsset:   true,
			checksumStatus: http.StatusOK,
		},
		{
			name:           "single space separator",
			checksumData:   `7b64d5722e18e5802335e87b06617702b957f6862549f5054a768023e74dd43b slackdump_Linux.tar.gz`,
			assetName:      "slackdump_Linux.tar.gz",
			wantHash:       "7b64d5722e18e5802335e87b06617702b957f6862549f5054a768023e74dd43b",
			wantErr:        false,
			includeAsset:   true,
			checksumStatus: http.StatusOK,
		},
		{
			name:           "multiple spaces separator",
			checksumData:   `7b64d5722e18e5802335e87b06617702b957f6862549f5054a768023e74dd43b     slackdump_Linux.tar.gz`,
			assetName:      "slackdump_Linux.tar.gz",
			wantHash:       "7b64d5722e18e5802335e87b06617702b957f6862549f5054a768023e74dd43b",
			wantErr:        false,
			includeAsset:   true,
			checksumStatus: http.StatusOK,
		},
		{
			name: "empty lines and whitespace",
			checksumData: `
7b64d5722e18e5802335e87b06617702b957f6862549f5054a768023e74dd43b  slackdump_Linux.tar.gz

298254e5604e5ce218ea026d25d52a4358f2542ed04d52671d4e806931ee2f49  slackdump_Windows.zip
`,
			assetName:      "slackdump_Windows.zip",
			wantHash:       "298254e5604e5ce218ea026d25d52a4358f2542ed04d52671d4e806931ee2f49",
			wantErr:        false,
			includeAsset:   true,
			checksumStatus: http.StatusOK,
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

func TestReplaceBinary(t *testing.T) {
	tests := []struct {
		name           string
		osName         string
		archiveType    string            // "zip" or "tar.gz"
		archiveContent map[string]string // filename -> content
		wantErr        bool
		errContains    string
		expectBinaryIn string // expected binary name in archive
	}{
		{
			name:        "successful replacement - linux tar.gz",
			osName:      "linux",
			archiveType: "tar.gz",
			archiveContent: map[string]string{
				"slackdump": "fake binary content",
			},
			wantErr:        false,
			expectBinaryIn: "slackdump",
		},
		{
			name:        "successful replacement - darwin tar.gz",
			osName:      "darwin",
			archiveType: "tar.gz",
			archiveContent: map[string]string{
				"slackdump": "fake binary content for mac",
			},
			wantErr:        false,
			expectBinaryIn: "slackdump",
		},
		{
			name:        "successful replacement - windows zip",
			osName:      "windows",
			archiveType: "zip",
			archiveContent: map[string]string{
				"slackdump.exe": "fake binary content for windows",
			},
			wantErr:        false,
			expectBinaryIn: "slackdump.exe",
		},
		{
			name:        "binary in subdirectory - tar.gz",
			osName:      "linux",
			archiveType: "tar.gz",
			archiveContent: map[string]string{
				"dist/slackdump": "fake binary in subdirectory",
			},
			wantErr:        false,
			expectBinaryIn: "dist/slackdump",
		},
		{
			name:        "binary in subdirectory - zip",
			osName:      "windows",
			archiveType: "zip",
			archiveContent: map[string]string{
				"dist/slackdump.exe": "fake windows binary in subdirectory",
			},
			wantErr:        false,
			expectBinaryIn: "dist/slackdump.exe",
		},
		{
			name:        "binary not found in tar.gz",
			osName:      "linux",
			archiveType: "tar.gz",
			archiveContent: map[string]string{
				"README.md": "some readme",
				"other.txt": "other file",
			},
			wantErr:     true,
			errContains: "binary not found in tar.gz archive",
		},
		{
			name:        "binary not found in zip",
			osName:      "windows",
			archiveType: "zip",
			archiveContent: map[string]string{
				"README.md": "some readme",
				"other.txt": "other file",
			},
			wantErr:     true,
			errContains: "binary not found in zip archive",
		},
		{
			name:           "empty tar.gz",
			osName:         "linux",
			archiveType:    "tar.gz",
			archiveContent: map[string]string{},
			wantErr:        true,
			errContains:    "binary not found in tar.gz archive",
		},
		{
			name:           "empty zip",
			osName:         "windows",
			archiveType:    "zip",
			archiveContent: map[string]string{},
			wantErr:        true,
			errContains:    "binary not found in zip archive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create a temporary directory for test files
			tmpDir := t.TempDir()

			// Create a test archive file
			var archivePath string
			var err error
			if tt.archiveType == "tar.gz" {
				archivePath = filepath.Join(tmpDir, "test.tar.gz")
				err = createTestTarGz(archivePath, tt.archiveContent)
			} else {
				archivePath = filepath.Join(tmpDir, "test.zip")
				err = createTestZip(archivePath, tt.archiveContent)
			}
			if err != nil {
				t.Fatalf("Failed to create test archive: %v", err)
			}

			// Create a fake existing executable
			exePath := filepath.Join(tmpDir, "slackdump-current")
			if err := os.WriteFile(exePath, []byte("old binary content"), 0755); err != nil {
				t.Fatalf("Failed to create test executable: %v", err)
			}

			// Run replaceBinary
			err = replaceBinary(ctx, archivePath, exePath, tt.osName)

			// Check error expectations
			if (err != nil) != tt.wantErr {
				t.Errorf("replaceBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("replaceBinary() error = %v, should contain %q", err, tt.errContains)
				}
				return
			}

			// If no error, verify the binary was replaced
			newContent, err := os.ReadFile(exePath)
			if err != nil {
				t.Errorf("Failed to read replaced binary: %v", err)
				return
			}

			expectedContent := tt.archiveContent[tt.expectBinaryIn]
			if string(newContent) != expectedContent {
				t.Errorf("Binary content mismatch: got %q, want %q", string(newContent), expectedContent)
			}

			// Verify the binary is executable on Unix-like systems
			// Note: We check both runtime.GOOS and tt.osName because:
			// 1. runtime.GOOS tells us the actual OS running the test
			// 2. tt.osName is the simulated OS for the test case
			// Windows doesn't support Unix permission bits, so we skip this check
			// when running on Windows, regardless of which OS we're simulating.
			if runtime.GOOS != "windows" && tt.osName != "windows" {
				info, err := os.Stat(exePath)
				if err != nil {
					t.Errorf("Failed to stat replaced binary: %v", err)
					return
				}
				if info.Mode().Perm()&0111 == 0 {
					t.Errorf("Binary is not executable: mode = %v", info.Mode())
				}
			}

			// Verify backup was removed
			backupPath := exePath + ".bak"
			if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
				t.Errorf("Backup file still exists at %s", backupPath)
			}
		})
	}
}

// createTestZip creates a zip file with the given contents for testing.
func createTestZip(zipPath string, contents map[string]string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			slog.Warn("failed to close file", "path", zipPath, "error", cerr)
		}
	}()

	zw := zip.NewWriter(f)
	defer func() {
		if cerr := zw.Close(); cerr != nil {
			slog.Warn("failed to close zip writer", "path", zipPath, "error", cerr)
		}
	}()

	for name, content := range contents {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(w, content); err != nil {
			return err
		}
	}

	return nil
}

// createTestTarGz creates a tar.gz file with the given contents for testing.
func createTestTarGz(tarGzPath string, contents map[string]string) error {
	f, err := os.Create(tarGzPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			slog.Warn("failed to close file", "path", tarGzPath, "error", cerr)
		}
	}()

	// Create gzip writer
	gzw := gzip.NewWriter(f)
	defer func() {
		if cerr := gzw.Close(); cerr != nil {
			slog.Warn("failed to close gzip writer", "path", tarGzPath, "error", cerr)
		}
	}()

	// Create tar writer
	tw := tar.NewWriter(gzw)
	defer func() {
		if cerr := tw.Close(); cerr != nil {
			slog.Warn("failed to close tar writer", "path", tarGzPath, "error", cerr)
		}
	}()

	for name, content := range contents {
		hdr := &tar.Header{
			Name: name,
			Mode: 0755,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}

func TestReplaceBinaryErrorCases(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	t.Run("invalid zip file", func(t *testing.T) {
		// Create an invalid zip file
		invalidZip := filepath.Join(tmpDir, "invalid.zip")
		if err := os.WriteFile(invalidZip, []byte("not a zip file"), 0644); err != nil {
			t.Fatalf("Failed to create invalid zip: %v", err)
		}

		exePath := filepath.Join(tmpDir, "fake-exe")
		if err := os.WriteFile(exePath, []byte("old"), 0755); err != nil {
			t.Fatalf("Failed to create exe: %v", err)
		}

		err := replaceBinary(ctx, invalidZip, exePath, "windows")
		if err == nil {
			t.Error("Expected error for invalid zip file")
		}
		if !strings.Contains(err.Error(), "failed to open zip") {
			t.Errorf("Expected 'failed to open zip' error, got: %v", err)
		}
	})

	t.Run("invalid tar.gz file", func(t *testing.T) {
		// Create an invalid tar.gz file
		invalidTarGz := filepath.Join(tmpDir, "invalid.tar.gz")
		if err := os.WriteFile(invalidTarGz, []byte("not a tar.gz file"), 0644); err != nil {
			t.Fatalf("Failed to create invalid tar.gz: %v", err)
		}

		exePath := filepath.Join(tmpDir, "fake-exe-tgz")
		if err := os.WriteFile(exePath, []byte("old"), 0755); err != nil {
			t.Fatalf("Failed to create exe: %v", err)
		}

		err := replaceBinary(ctx, invalidTarGz, exePath, "linux")
		if err == nil {
			t.Error("Expected error for invalid tar.gz file")
		}
		if !strings.Contains(err.Error(), "failed to create gzip reader") {
			t.Errorf("Expected 'failed to create gzip reader' error, got: %v", err)
		}
	})

	t.Run("unsupported archive format", func(t *testing.T) {
		// Create a file with unsupported extension
		unsupportedFile := filepath.Join(tmpDir, "archive.rar")
		if err := os.WriteFile(unsupportedFile, []byte("some content"), 0644); err != nil {
			t.Fatalf("Failed to create unsupported file: %v", err)
		}

		exePath := filepath.Join(tmpDir, "fake-exe-rar")
		if err := os.WriteFile(exePath, []byte("old"), 0755); err != nil {
			t.Fatalf("Failed to create exe: %v", err)
		}

		err := replaceBinary(ctx, unsupportedFile, exePath, "linux")
		if err == nil {
			t.Error("Expected error for unsupported archive format")
		}
		if !strings.Contains(err.Error(), "unsupported archive format") {
			t.Errorf("Expected 'unsupported archive format' error, got: %v", err)
		}
	})

	t.Run("nonexistent zip file", func(t *testing.T) {
		exePath := filepath.Join(tmpDir, "fake-exe2")
		if err := os.WriteFile(exePath, []byte("old"), 0755); err != nil {
			t.Fatalf("Failed to create exe: %v", err)
		}

		err := replaceBinary(ctx, "/nonexistent/file.zip", exePath, "windows")
		if err == nil {
			t.Error("Expected error for nonexistent zip file")
		}
	})

	t.Run("nonexistent tar.gz file", func(t *testing.T) {
		exePath := filepath.Join(tmpDir, "fake-exe3")
		if err := os.WriteFile(exePath, []byte("old"), 0755); err != nil {
			t.Fatalf("Failed to create exe: %v", err)
		}

		err := replaceBinary(ctx, "/nonexistent/file.tar.gz", exePath, "linux")
		if err == nil {
			t.Error("Expected error for nonexistent tar.gz file")
		}
	})
}
