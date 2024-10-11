package info

import (
	"runtime"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
)

type SlackdumpInfo struct {
	Build     cfg.BuildInfo `json:"build"`
	GoVersion string        `json:"go_version"`
}

func (inf *SlackdumpInfo) collect(PathReplFunc) {
	inf.Build = cfg.Version
	inf.GoVersion = runtime.Version()
}
