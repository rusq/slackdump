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
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
