package list

import (
	"context"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var CmdList = &base.Command{
	Run:       runList,
	UsageLine: "slackdump list",
	Short:     "list users or channels",
	Long: `
List lists users or channels for the Slack Workspace.  It may take a while on
large workspaces, as Slack limits the amount of requests on it's own discretion,
which is sometimes unreasonably slow.
`,
	Commands: []*base.Command{
		CmdListUsers,
	},
}

func runList(ctx context.Context, cmd *base.Command, args []string) error {
	cmd.Flag.Usage()
	return nil
}
