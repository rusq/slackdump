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
// Package pwcompat provides a compatibility layer, so when the playwright-go
// team decides to break compatibility again, there's a place to write a
// workaround.
package pwcompat

import (
	"log"
	"os"
	"path/filepath"

	"github.com/playwright-community/playwright-go"

	"github.com/rusq/slackdump/v3/internal/structures"
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
	drvdir := filepath.Join(structures.NVL(runopts.DriverDirectory, cacheDir), "ms-playwright-go", drv.Version)
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
