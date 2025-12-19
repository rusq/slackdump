package workspace

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
)

var cmdWspSelect = &base.Command{
	UsageLine: baseCommand + " select [flags]",
	Short:     "choose a previously saved workspace",
	Long: `
# Workspace Select Command

**Select** allows to set the current workspace from the list of workspaces
that you have previously authenticated in.

"Current" means that this workspace will be used by default when running
other commands, unless you specify a different workspace explicitly with
the ` + "`-w`" + ` flag.

To get the full list of authenticated workspaces, run:

	` + base.Executable() + ` auth list
`,
	FlagMask:   flagmask,
	PrintFlags: true,
	Wizard:     wizSelect,
}

func init() {
	cmdWspSelect.Run = runSelect
}

func runSelect(ctx context.Context, cmd *base.Command, args []string) error {
	wsp := argsWorkspace(args, cfg.Workspace)
	if wsp == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return cache.ErrNameRequired
	}
	m, err := CacheMgr()
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return fmt.Errorf("unable to initialise cache: %s", err)
	}
	// TODO: maybe ask the user to create new workspace if the workspace
	// does not exist.
	if err := m.Select(wsp); err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("unable to select %q: %w", args[0], err)
	}
	fmt.Printf("Success:  current workspace set to:  %s\n", args[0])
	return nil
}
