package cfg

import (
	"os"
	"path/filepath"

	"github.com/rusq/dlog"
)

const (
	cacheDirName = "slackdump"
)

// ucd detects user cache dir and returns slack cache directory name.
func ucd(ucdFn func() (string, error)) string {
	ucd, err := ucdFn()
	if err != nil {
		dlog.Debug(err)
		return "."
	}
	return filepath.Join(ucd, cacheDirName)
}

func CacheDir() string {
	if SlackOptions.CacheDir == "" {
		return ucd(os.UserCacheDir)
	}
	return SlackOptions.CacheDir
}
