package info

import (
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/internal/cache"
)

type workspace struct {
	Path       string `json:"path"`
	TxtExists  bool   `json:"txt_exists"`
	HasDefault bool   `json:"has_default"`
	Count      int    `json:"count"`
}

func (inf *workspace) collect() {
	inf.Path = replaceFn(cfg.LocalCacheDir)
	inf.Count = -1
	// Workspace information
	m, err := cache.NewManager(cfg.LocalCacheDir)
	if err != nil {
		inf.Path = err.Error()
		return
	}
	if _, err := m.Current(); err == nil {
		inf.TxtExists = true
	}
	wsp, err := m.List()
	if err == nil {
		inf.Count = len(wsp)
	}
}
