package info

import (
	"os"

	"github.com/go-rod/rod/lib/launcher"
)

type rodinfo struct {
	Path     string   `json:"path"`
	Browsers []string `json:"browsers"`
}

func (inf *rodinfo) collect() {
	inf.Path = homerepl(launcher.DefaultBrowserDir)
	if de, err := os.ReadDir(launcher.DefaultBrowserDir); err == nil {
		inf.Browsers = dirnames(de)
	}
}
