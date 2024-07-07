// Package pwcompat provides a compatibility layer, so when the playwright-go
// team decides to break compatibility again, there's a place to write a
// workaround.
package pwcompat

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/playwright-community/playwright-go"
)

// Workaround for unexported driver dir in playwright.

// newDriverFn is the function that creates a new driver.  It is set to
// playwright.NewDriver by default, but can be overridden for testing.
var newDriverFn = playwright.NewDriver

func getDefaultCacheDirectory() (string, error) {
	// pinched from playwright
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home directory: %w", err)
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(userHomeDir, "AppData", "Local"), nil
	case "darwin":
		return filepath.Join(userHomeDir, "Library", "Caches"), nil
	case "linux":
		return filepath.Join(userHomeDir, ".cache"), nil
	}
	return "", errors.New("could not determine cache directory")
}

func NewDriver(runopts *playwright.RunOptions) (*playwright.PlaywrightDriver, error) {
	drv, err := newDriverFn(runopts)
	if err != nil {
		return nil, fmt.Errorf("error initialising driver: %w", err)
	}
	return drv, nil
}

// DriverDir returns the driver directory, broken in this commit:
// https://github.com/playwright-community/playwright-go/pull/449/commits/372e209c776222f4681cf1b24a1379e3648dd982
func DriverDir(runopts *playwright.RunOptions) (string, error) {
	drv, err := NewDriver(runopts)
	if err != nil {
		return "", err
	}
	baseDriverDirectory, err := getDefaultCacheDirectory()
	if err != nil {
		return "", fmt.Errorf("it's just not your day: %w", err)
	}
	driverDirectory := filepath.Join(nvl(runopts.DriverDirectory, baseDriverDirectory), "ms-playwright-go", drv.Version)
	return driverDirectory, nil
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
