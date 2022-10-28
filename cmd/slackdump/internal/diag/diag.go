package diag

import (
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

// CmdDiag is the diagnostic tool.
var CmdDiag = &base.Command{
	Run:       nil,
	Wizard:    nil,
	UsageLine: "slackdump diag",
	Short:     "problem diagnostic tools",
	Long: `
Diag contains various diagnostic tool, running which may be requested if you
open an issue on Github.
`,
	CustomFlags: false,
	FlagMask:    0,
	PrintFlags:  false,
	RequireAuth: false,
	Commands: []*base.Command{
		CmdRawOutput,
		CmdEzTest,
		CmdThread,
	},
}
