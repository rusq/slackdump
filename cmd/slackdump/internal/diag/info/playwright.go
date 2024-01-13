package info

import (
	"os"
	"path/filepath"

	"github.com/playwright-community/playwright-go"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
)

type PwInfo struct {
	Path              string   `json:"path"`
	InstalledVersions []string `json:"installed_versions"`
	InstalledBrowsers []string `json:"installed_browsers"`
	Script            string   `json:"script"`
	ScriptExists      bool     `json:"script_exists"`
	ScriptPerm        string   `json:"script_perm"`
}

func (inf *PwInfo) collect() {
	pwdrv, err := playwright.NewDriver(&playwright.RunOptions{
		Browsers:            []string{cfg.Browser.String()},
		SkipInstallBrowsers: true},
	)
	if err != nil {
		inf.Path = looser(err)
		return
	}
	inf.Path = replaceFn(pwdrv.DriverDirectory)
	inf.Script = replaceFn(pwdrv.DriverBinaryLocation)
	if inf.Script != "" {
		if stat, err := os.Stat(pwdrv.DriverBinaryLocation); err == nil {
			inf.ScriptPerm = stat.Mode().String()
			inf.ScriptExists = true
		} else {
			inf.ScriptPerm = looser(err)
		}
	}
	if de, err := os.ReadDir(filepath.Join(pwdrv.DriverDirectory, "..")); err == nil {
		inf.InstalledVersions = dirnames(de)
	} else {
		inf.InstalledVersions = []string{looser(err)}
	}

	if de, err := os.ReadDir(filepath.Join(pwdrv.DriverDirectory, "..", "..", "ms-playwright")); err == nil {
		inf.InstalledBrowsers = dirnames(de)
	} else {
		inf.InstalledBrowsers = []string{looser(err)}
	}
}
