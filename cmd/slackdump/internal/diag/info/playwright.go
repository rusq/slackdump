package info

import (
	"os"
	"path/filepath"

	"github.com/playwright-community/playwright-go"

	"github.com/rusq/slackdump/v3/auth/browser/pwcompat"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace/wspcfg"
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
