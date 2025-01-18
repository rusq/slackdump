package info

import (
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace"
)

type Workspace struct {
	Path       string `json:"path"`
	TxtExists  bool   `json:"txt_exists"`
	HasDefault bool   `json:"has_default"`
	Count      int    `json:"count"`
}

func (inf *Workspace) collect(replaceFn PathReplFunc) {
	inf.Path = replaceFn(cfg.CacheDir())
	inf.Count = -1
	// Workspace information
	m, err := workspace.CacheMgr()
	if err != nil {
		inf.Path = loser(err)
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
