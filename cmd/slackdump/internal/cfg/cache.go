package cfg

import (
	"log/slog"
	"os"
	"path/filepath"
)

const (
	cacheDirName = "slackdump"
)

// ucd detects user cache dir and returns slack cache directory name.
func ucd(ucdFn func() (string, error)) string {
	ucd, err := ucdFn()
	if err != nil {
		slog.Debug("ucd", "error", err)
		return "."
	}
	return filepath.Join(ucd, cacheDirName)
}

func CacheDir() string {
	if LocalCacheDir == "" {
		return ucd(os.UserCacheDir)
	}
	return LocalCacheDir
}
