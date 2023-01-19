package list

import (
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/dump"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

const codeBlock = "```"

// CmdListConversation is the command to list conversations.
var CmdListConversation = &base.Command{
	UsageLine:   "slackdump list convo [flags] <conversation list>",
	Short:       "synonym for 'slackdump dump'",
	PrintFlags:  true,
	RequireAuth: true,
	FlagMask:    dump.CmdDump.FlagMask,
	Long: base.Render(`
# List Conversation Command

This command does effectvely the same as the following: 
` + codeBlock + `
slackdump dump
` + codeBlock + `

To get extended usage help, run ` + "`slackdump help dump`",
	)}

func init() {
	CmdListConversation.Run = dump.RunDump
	dump.InitDumpFlagset(&CmdListConversation.Flag)
}
