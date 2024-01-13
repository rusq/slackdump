package info

import (
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/internal/cache"
)

type EZLogin struct {
	Flags   map[string]bool `json:"flags"`
	Browser string          `json:"browser"`
}

func (inf *EZLogin) collect() {
	inf.Flags = cache.EzLoginFlags()
	inf.Browser = cfg.Browser.String()
}
