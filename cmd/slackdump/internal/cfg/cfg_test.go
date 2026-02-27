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

// Package cfg contains common configuration variables.
package cfg

import (
	"errors"
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetBaseFlags(t *testing.T) {
	t.Run("all flags are set", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ExitOnError)
		mask := DefaultFlags

		SetBaseFlags(fs, mask)

		// Test flag parsing and assignment
		err := fs.Parse([]string{
			"-trace", "trace.log",
			"-log", "log.txt",
			"-v",
			"-enterprise",
			"-files=false",
			"-api-config", "config.json",
			"-o", "output.zip",
			"-cache-dir", "/tmp/cache",
			"-workspace", "my_workspace",
			"-no-user-cache",
			"-user-cache-retention", "30m",
			"-no-chunk-cache",
			"-time-from", "2022-01-01T00:00:00",
			"-time-to", "2022-01-31T23:59:59",
		})
		if err != nil {
			t.Fatalf("Error parsing flags: %v", err)
		}

		// Test flag values
		if TraceFile != "trace.log" {
			t.Errorf("Expected TraceFile to be 'trace.log', got '%s'", TraceFile)
		}
		if LogFile != "log.txt" {
			t.Errorf("Expected LogFile to be 'log.txt', got '%s'", LogFile)
		}
		if !Verbose {
			t.Error("Expected Verbose to be true, got false")
		}
		if !ForceEnterprise {
			t.Error("Expected ForceEnterprise to be true, got false")
		}
		if WithFiles {
			t.Error("Expected DownloadFiles to be false, got true")
		}
		if ConfigFile != "config.json" {
			t.Errorf("Expected ConfigFile to be 'config.json', got '%s'", ConfigFile)
		}
		if Output != "output.zip" {
			t.Errorf("Expected Output to be 'output.zip', got '%s'", Output)
		}
		if LocalCacheDir != "/tmp/cache" {
			t.Errorf("Expected LocalCacheDir to be '/tmp/cache', got '%s'", LocalCacheDir)
		}
		if Workspace != "my_workspace" {
			t.Errorf("Expected Workspace to be 'my_workspace', got '%s'", Workspace)
		}
		if !NoUserCache {
			t.Error("Expected NoUserCache to be true, got false")
		}
		if UserCacheRetention != 30*time.Minute {
			t.Errorf("Expected UserCacheRetention to be 30 minutes, got %s", UserCacheRetention)
		}
		if !NoChunkCache {
			t.Error("Expected NoChunkCache to be true, got false")
		}
		if Oldest.String() != "2022-01-01" {
			t.Errorf("Expected Oldest to be '2022-01-01', got '%s'", Oldest.String())
		}
		if Latest.String() != "2022-01-31T23:59:59" {
			t.Errorf("Expected Latest to be '2022-01-31T23:59:59Z', got '%s'", Latest.String())
		}
	})
	t.Run("omit cache dir set", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ExitOnError)
		mask := OmitCacheDir

		SetBaseFlags(fs, mask)
		fs.Parse([]string{})

		assert.Equal(t, LocalCacheDir, CacheDir())
	})
}

func TestBuildInfo_IsReleased(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{
			name:     "valid release version",
			version:  "v3.2.1",
			expected: true,
		},
		{
			name:     "another valid release version",
			version:  "v1.0.0",
			expected: true,
		},
		{
			name:     "empty version",
			version:  "",
			expected: false,
		},
		{
			name:     "unknown version",
			version:  "unknown",
			expected: false,
		},
		{
			name:     "version without v. prefix",
			version:  "3.2.1",
			expected: true,
		},
		{
			name:     "development version",
			version:  "dev",
			expected: false,
		},
		{
			name:     "v-prefixed development marker",
			version:  "vdev",
			expected: false,
		},
		{
			name:     "v-prefixed alpha marker",
			version:  "valpha",
			expected: false,
		},
		{
			name:     "v-prefixed test marker",
			version:  "vtest",
			expected: false,
		},
		{
			name:     "just v prefix without version",
			version:  "v",
			expected: false,
		},
		{
			name:     "capital V unknown version",
			version:  "Vunknown",
			expected: false,
		},
		{
			name:     "capital V valid version",
			version:  "V2.3.4",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bi := BuildInfo{
				Version: tt.version,
				Commit:  "abc123",
				Date:    "2024-01-01",
			}
			assert.Equal(t, tt.expected, bi.IsReleased())
		})
	}
}

func TestBuildInfo_Normalised(t *testing.T) {
	const (
		commit     = "deadbeef"
		makeDate   = "2026-02-25 10:00:00Z"
		brewDate   = "2026-02-25T10:00:00Z"
		normalDate = "2026-02-25 10:00:00Z" // canonical output (make layout)
	)

	tests := []struct {
		name        string
		input       BuildInfo
		wantVersion string
		wantDate    string
		wantCommit  string
		wantErr     error
	}{
		{
			name:        "lowercase v prefix with make date",
			input:       BuildInfo{Version: "v3.2.1", Commit: commit, Date: makeDate},
			wantVersion: "v3.2.1",
			wantDate:    normalDate,
			wantCommit:  commit,
		},
		{
			name:        "no v prefix (homebrew style) with make date",
			input:       BuildInfo{Version: "3.2.1", Commit: "Homebrew", Date: makeDate},
			wantVersion: "v3.2.1",
			wantDate:    normalDate,
			wantCommit:  "Homebrew",
		},
		{
			name:        "homebrew T-format date is normalised",
			input:       BuildInfo{Version: "4.0.0", Commit: "Homebrew", Date: brewDate},
			wantVersion: "v4.0.0",
			wantDate:    normalDate,
			wantCommit:  "Homebrew",
		},
		{
			name:        "capital V prefix is lowercased",
			input:       BuildInfo{Version: "V2.3.4", Commit: commit, Date: makeDate},
			wantVersion: "v2.3.4",
			wantDate:    normalDate,
			wantCommit:  commit,
		},
		{
			name:    "unreleased dev version returns ErrVerUnknown",
			input:   BuildInfo{Version: "dev", Commit: commit, Date: makeDate},
			wantErr: ErrVerUnknown,
		},
		{
			name:    "unknown version string returns ErrVerUnknown",
			input:   BuildInfo{Version: "unknown", Commit: commit, Date: makeDate},
			wantErr: ErrVerUnknown,
		},
		{
			name:    "empty version returns ErrVerUnknown",
			input:   BuildInfo{Version: "", Commit: commit, Date: makeDate},
			wantErr: ErrVerUnknown,
		},
		{
			name:    "empty date returns ErrDateUnknown",
			input:   BuildInfo{Version: "v1.0.0", Commit: commit, Date: ""},
			wantErr: ErrDateUnknown,
		},
		{
			name:    "unknown date string returns ErrDateUnknown",
			input:   BuildInfo{Version: "v1.0.0", Commit: commit, Date: "unknown"},
			wantErr: ErrDateUnknown,
		},
		{
			name:    "unrecognised date format returns ErrDateUnknown",
			input:   BuildInfo{Version: "v1.0.0", Commit: commit, Date: "25/02/2026"},
			wantErr: ErrDateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.Normalised()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "expected error %v, got %v", tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, got.Version)
			assert.Equal(t, tt.wantDate, got.Date)
			assert.Equal(t, tt.wantCommit, got.Commit)
		})
	}
}
