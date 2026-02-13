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

package info

import (
	"os"
	"path/filepath"

	"github.com/playwright-community/playwright-go"

	"github.com/rusq/slackdump/v4/auth/browser/pwcompat"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/workspace/wspcfg"
)

type PwInfo struct {
	Path              string   `json:"path"`
	InstalledVersions []string `json:"installed_versions"`
	BrowsersPath      string
	InstalledBrowsers []string `json:"installed_browsers"`
	Script            string   `json:"script"`
	ScriptExists      bool     `json:"script_exists"`
	ScriptPerm        string   `json:"script_perm"`
}

func (inf *PwInfo) collect(replaceFn PathReplFunc) {
	opts := &playwright.RunOptions{
		Browsers:            []string{wspcfg.Browser.String()},
		SkipInstallBrowsers: true,
	}
	pwdrv, err := pwcompat.NewAdapter(opts)
	if err != nil {
		inf.Path = loser(err)
		return
	}

	inf.Path = replaceFn(pwdrv.DriverDirectory)
	inf.Script = replaceFn(pwdrv.DriverBinaryLocation)
	if inf.Script != "" {
		if stat, err := os.Stat(pwdrv.DriverBinaryLocation); err == nil {
			inf.ScriptPerm = stat.Mode().String()
			inf.ScriptExists = true
		} else {
			inf.ScriptPerm = loser(err)
		}
	}
	if de, err := os.ReadDir(filepath.Join(pwdrv.DriverDirectory, "..")); err == nil {
		inf.InstalledVersions = dirnames(de)
	} else {
		inf.InstalledVersions = []string{loser(err)}
	}

	browserPath := filepath.Join(pwdrv.DriverDirectory, "..", "..", "ms-playwright")
	inf.BrowsersPath = replaceFn(browserPath)

	if de, err := os.ReadDir(browserPath); err == nil {
		inf.InstalledBrowsers = dirnames(de)
	} else {
		inf.InstalledBrowsers = []string{loser(err)}
	}
}
