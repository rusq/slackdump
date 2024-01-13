package info

import (
	"os"
	"runtime"

	"github.com/rusq/slackdump/v2/auth"
)

type osinfo struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	IsDocker  bool   `json:"is_docker"`
	IsXactive bool   `json:"is_x_active"`
}

func (inf *osinfo) collect() {
	inf.OS = runtime.GOOS
	inf.Arch = runtime.GOARCH
	inf.IsDocker = auth.IsDocker()
	inf.IsXactive = os.Getenv("DISPLAY") != ""
}
