package dump

import (
	_ "embed"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

//go:embed assets/dump.md
var dumpLong string

var CmdDump = &base.Command{
	UsageLine: "slackdump dump [flags] <IDs or URLs>",
	Short:     "dump individual conversations or threads",
	Long:      base.Render(dumpLong),
}
