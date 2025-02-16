package info

import (
	"os"
	"runtime"

	"github.com/rusq/slackdump/v3/internal/osext"
)

type OSInfo struct {
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	IsDocker      bool   `json:"is_docker"`
	IsXactive     bool   `json:"is_x_active"`
	IsInteractive bool   `json:"is_interactive"`
}

func (inf *OSInfo) collect(PathReplFunc) {
	inf.OS = runtime.GOOS
	inf.Arch = runtime.GOARCH
	inf.IsDocker = osext.IsDocker()
	inf.IsXactive = os.Getenv("DISPLAY") != ""
	inf.IsInteractive = osext.IsInteractive()
}
