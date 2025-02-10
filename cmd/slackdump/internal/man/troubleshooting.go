package man

import (
	_ "embed"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

//go:embed assets/troubleshooting.md
var troubleshootingMd string

var Troubleshooting = &base.Command{
	UsageLine: "slackdump help troubleshooting",
	Short:     "troubleshooting related information",
	Long:      troubleshootingMd,
}
