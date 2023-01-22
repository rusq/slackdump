package dump

import (
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/list"
)

const codeBlock = "```"

// CmdDump is the dump command.
var CmdDump = &base.Command{
	UsageLine: "slackdump dump [flags] <IDs or URLs>",
	Short:     "dump individual conversations or threads",
	Long: base.Render(`
	# Dump Command

This command is an alias for: 
` + codeBlock + `
slackdump list conversation
` + codeBlock + `

To get extended usage help, run ` + "`slackdump help list conversation`"),
	RequireAuth: true,
	PrintFlags:  true,
}

func init() {
	CmdDump.Run = list.RunDump
	CmdDump.Wizard = list.WizDump
	CmdDump.Long = list.HelpDump(CmdDump)
}
