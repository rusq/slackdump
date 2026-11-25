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
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mxschmitt/playwright-go"

	"github.com/rusq/slackdump/v4/internal/structures"
)

// Install installs Playwright, recovering an incomplete managed driver download.
func Install(runopts *playwright.RunOptions) error {
	drv, err := prepareDriver(runopts)
	if err != nil {
		return err
	}
	if err := drv.Install(); err != nil {
		return fmt.Errorf("could not install driver: %w", err)
	}
	return nil
}

func prepareDriver(runopts *playwright.RunOptions) (*playwright.PlaywrightDriver, error) {
	opts := *runopts // Playwright fills in defaults in place.
	managed := opts.DriverDirectory == "" && os.Getenv("PLAYWRIGHT_DRIVER_PATH") == "" &&
		os.Getenv("PLAYWRIGHT_CLI_PATH") == "" && os.Getenv("PLAYWRIGHT_NODEJS_PATH") == ""
	drv, err := playwright.NewDriver(&opts)
	if err != nil {
		return nil, fmt.Errorf("could not get driver instance: %w", err)
	}
	if !managed {
		return drv, nil
	}
	if _, err := os.Lstat(drv.Command().Path); !errors.Is(err, os.ErrNotExist) {
		return drv, nil
	}

	// Playwright extracts cli.js before downloading Node. If that download
	// fails, Playwright tries to run the missing Node executable on the next
	// install. Removing the marker makes Playwright download the driver again.
	packageDir := filepath.Join(opts.DriverDirectory, "package")
	// A link inside the user cache can point to an externally managed install.
	for _, dir := range []string{filepath.Dir(opts.DriverDirectory), opts.DriverDirectory, packageDir} {
		if info, err := os.Lstat(dir); err != nil || !info.IsDir() {
			return drv, nil
		}
	}
	cliPath := filepath.Join(packageDir, "cli.js")
	if info, err := os.Lstat(cliPath); err != nil || !info.Mode().IsRegular() {
		return drv, nil
	}
	if err := os.Remove(cliPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("could not clear incomplete Playwright driver: %w", err)
	}
	return drv, nil
}

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
