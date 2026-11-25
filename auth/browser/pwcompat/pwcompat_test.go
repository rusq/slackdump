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

package pwcompat

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
)

func Test_prepareDriver(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
	default:
		t.Skip("Playwright has no default cache directory on this platform")
	}
	const driverPath = "PLAYWRIGHT_DRIVER_PATH"
	const cliPath = "PLAYWRIGHT_CLI_PATH"
	const nodePath = "PLAYWRIGHT_NODEJS_PATH"
	for _, key := range []string{driverPath, cliPath, nodePath} {
		t.Setenv(key, "")
	}

	for _, tt := range []struct {
		name           string
		hasNode        bool
		explicitDriver bool
		override       string
		link           string
		wantCleared    bool
	}{
		{name: "incomplete managed driver", wantCleared: true},
		{name: "managed driver with Node", hasNode: true},
		{name: "explicit driver directory", explicitDriver: true},
		{name: "driver directory override", override: driverPath},
		{name: "CLI override", override: cliPath},
		{name: "Node override", override: nodePath},
		{name: "linked Playwright cache directory", link: "playwright"},
		{name: "linked driver version directory", link: "version"},
		{name: "linked driver package directory", link: "package"},
		{name: "linked user cache directory", link: "cache", wantCleared: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			defaults := &playwright.RunOptions{}
			drv, err := playwright.NewDriver(defaults)
			require.NoError(t, err)

			var link string
			switch tt.link {
			case "playwright":
				link = filepath.Dir(defaults.DriverDirectory)
			case "version":
				link = defaults.DriverDirectory
			case "package":
				link = filepath.Join(defaults.DriverDirectory, "package")
			case "cache":
				link = filepath.Dir(filepath.Dir(defaults.DriverDirectory))
			}
			if link != "" {
				require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
				err := os.Symlink(t.TempDir(), link)
				if err != nil && runtime.GOOS == "windows" {
					t.Skipf("cannot create test symlink: %v", err)
				}
				require.NoError(t, err)
			}

			cli := writeCLI(t, defaults.DriverDirectory)
			other := filepath.Join(defaults.DriverDirectory, "package", "other.js")
			require.NoError(t, os.WriteFile(other, nil, 0o600))
			if tt.hasNode {
				require.NoError(t, os.WriteFile(drv.Command().Path, nil, 0o600))
			}

			opts := &playwright.RunOptions{}
			var customCLI string
			if tt.explicitDriver || tt.override != "" {
				customDir := t.TempDir()
				customCLI = writeCLI(t, customDir)
				switch tt.override {
				case driverPath:
					t.Setenv(driverPath, customDir)
				case cliPath:
					t.Setenv(cliPath, customCLI)
				case nodePath:
					t.Setenv(nodePath, filepath.Join(customDir, NodeExe))
				default:
					opts.DriverDirectory = customDir
				}
			}

			originalDriverDir := opts.DriverDirectory
			_, err = prepareDriver(opts)
			require.NoError(t, err)
			require.Equal(t, originalDriverDir, opts.DriverDirectory)
			if tt.wantCleared {
				require.NoFileExists(t, cli)
			} else {
				require.FileExists(t, cli)
			}
			require.FileExists(t, other)
			if customCLI != "" {
				require.FileExists(t, customCLI)
			}
		})
	}
}

func writeCLI(t *testing.T, driverDir string) string {
	t.Helper()
	cli := filepath.Join(driverDir, "package", "cli.js")
	require.NoError(t, os.MkdirAll(filepath.Dir(cli), 0o755))
	require.NoError(t, os.WriteFile(cli, nil, 0o600))
	return cli
}
