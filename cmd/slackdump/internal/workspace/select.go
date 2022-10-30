package workspace

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/app/appauth"
)

var CmdWspSelect = &base.Command{
	UsageLine: "slackdump workspace select [flags]",
	Short:     "choose a previously saved workspace",
	Long: `
Select allows to set the current workspace from the list of workspaces
that you have previously authenticated in.

To get the full list of workspaces, run:

	` + base.Executable() + ` workspace list
`,
	FlagMask:   flagmask,
	PrintFlags: true,
}

func init() {
	CmdWspSelect.Run = runSelect
}

func runSelect(ctx context.Context, cmd *base.Command, args []string) {
	if len(args) == 0 {
		base.SetExitStatusMsg(base.SInvalidParameters, "workspace name is not specified")
		return
	}
	m, err := appauth.NewManager(cfg.CacheDir())
	if err != nil {
		base.SetExitStatusMsg(base.SCacheError, fmt.Sprintf("unable to initialise cache: %s", err))
		return
	}
	if err := m.Select(args[0]); err != nil {
		base.SetExitStatusMsg(base.SInvalidParameters, fmt.Sprintf("Failed:  unable to select %s: %s", args[0], err))
		return
	}
	fmt.Printf("Success:  current workspace set to:  %s\n", args[0])
}
