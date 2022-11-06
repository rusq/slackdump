package dump

import "github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"

var CmdDump = &base.Command{
	UsageLine: "slackdump dump [flags] <ids or urls>",
	Short:     "dump individual conversations or threads",
	Long: base.Render(`
# Command Dump

Dump is the mode that allows to dump the following type of conversations:
- public and private channels with threads
- group messages (MPIM)
- private messages (DMs)
- individual threads

It downloads file attachments as well.

This is the original mode of the Slackdump, so its behaviour would be familiar 
to those who used it since the first release.

It can be considered as a low level mode, as there are no transformations, and
it is directly calling Slackdump library functions.
`),
}
