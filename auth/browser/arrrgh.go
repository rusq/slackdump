package browser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/playwright-community/playwright-go"
)

// Just because some motherfuck decided that unexporting DriverDirectory is a good idea.

func getDefaultCacheDirectory() (string, error) {
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

func pwDriverDir(runopts *playwright.RunOptions) (string, error) {
	drv, err := newDriverFn(runopts)
	if err != nil {
		return "", fmt.Errorf("error initialising driver: %w", err)
	}
	baseDriverDirectory, err := getDefaultCacheDirectory()
	if err != nil {
		return "", fmt.Errorf("it's just not your day: %w", err)
	}
	driverDirectory := filepath.Join(nvl(runopts.DriverDirectory, baseDriverDirectory), "ms-playwright-go", drv.Version)
	return driverDirectory, nil
}
