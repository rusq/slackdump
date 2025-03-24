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
