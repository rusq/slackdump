// Package pwcompat provides a compatibility layer, so when the playwright-go
// team decides to break compatibility again, there's a place to write a
// workaround.
package pwcompat

import (
	"log"
	"os"
	"path/filepath"

	"github.com/playwright-community/playwright-go"
)

// Workaround for unexported driver dir in playwright.

var (
	// environment related variables
	homedir  string = must(os.UserHomeDir())
	cacheDir string // platform dependent
	NodeExe  string = "node"
)

func must[T any](v T, e error) T {
	if e != nil {
		log.Panicf("error getting user home directory: %s", e)
	}
	return v
}

type Adapter struct {
	DriverDirectory      string
	DriverBinaryLocation string

	drv  *playwright.PlaywrightDriver
	opts *playwright.RunOptions
}

func NewAdapter(runopts *playwright.RunOptions) (*Adapter, error) {
	drv, err := playwright.NewDriver(runopts)
	if err != nil {
		return nil, err
	}
	if cacheDir == "" { // i.e. freebsd etc.
		cacheDir, _ = os.UserCacheDir()
	}
	drvdir := filepath.Join(nvl(runopts.DriverDirectory, cacheDir), "ms-playwright-go", drv.Version)
	drvbin := filepath.Join(drvdir, NodeExe)

	return &Adapter{
		drv:                  drv,
		opts:                 runopts,
		DriverDirectory:      drvdir,
		DriverBinaryLocation: drvbin,
	}, nil
}

func (a *Adapter) Driver() *playwright.PlaywrightDriver {
	return a.drv
}

func nvl(first string, rest ...string) string {
	if first != "" {
		return first
	}
	for _, s := range rest {
		if s != "" {
			return s
		}
	}
	return ""
}
