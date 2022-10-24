package list

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var CmdListUsers = &base.Command{
	Run:        listUsers,
	UsageLine:  "slackdump list users [flags]",
	PrintFlags: true,
	Short:      "list workspace users",
	Long: `
List users lists workspace users in the desired format.
`,
}

func init() {
	var dummy string
	CmdListUsers.Flag.StringVar(&dummy, "test", "test string", "this is a test string")
}

func listUsers(ctx context.Context, cmd *base.Command, args []string) {
	fmt.Println("list users invoked, args:", args)
}
