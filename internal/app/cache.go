package app

import (
	"os"
	"path/filepath"

	"github.com/rusq/dlog"
)

const (
	cacheDirName = "slackdump"
)

var cacheDir string // cache directory

func init() {
	var err error
	ucd, err := os.UserCacheDir()
	if err != nil {
		dlog.Debugf("failed to determine the OS cache directory: %s", err)
		return
	}
	cacheDir = filepath.Join(ucd, cacheDirName)
}

func CacheDir() string {
	return cacheDir
}
