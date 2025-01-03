package man

import (
	_ "embed"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

//go:embed assets/quickstart.md
var quickstartMD string

var Quickstart = &base.Command{
	UsageLine: "slackdump quickstart",
	Short:     "quickstart guide",
	Long:      quickstartMD,
}
