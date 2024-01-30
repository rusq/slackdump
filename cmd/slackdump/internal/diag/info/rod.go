package info

import (
	"os"

	"github.com/go-rod/rod/lib/launcher"
)

type RodInfo struct {
	Path     string   `json:"path"`
	Browsers []string `json:"browsers"`
}

func (inf *RodInfo) collect(replaceFn PathReplFunc) {
	inf.Path = replaceFn(launcher.DefaultBrowserDir)
	if de, err := os.ReadDir(launcher.DefaultBrowserDir); err == nil {
		inf.Browsers = dirnames(de)
	} else {
		inf.Browsers = []string{loser(err)}
	}
}
