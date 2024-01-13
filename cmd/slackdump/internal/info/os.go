package info

import (
	"runtime"

	"github.com/rusq/slackdump/v2/auth"
)

type osinfo struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	IsDocker bool   `json:"is_docker"`
}

func (inf *osinfo) collect() {
	inf.OS = runtime.GOOS
	inf.Arch = runtime.GOARCH
	inf.IsDocker = auth.IsDocker()
}
