package info

import (
	"os"
	"runtime"

	"github.com/rusq/slackdump/v3/auth"
)

type OSInfo struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	IsDocker  bool   `json:"is_docker"`
	IsXactive bool   `json:"is_x_active"`
}

func (inf *OSInfo) collect() {
	inf.OS = runtime.GOOS
	inf.Arch = runtime.GOARCH
	inf.IsDocker = auth.IsDocker()
	inf.IsXactive = os.Getenv("DISPLAY") != ""
}
