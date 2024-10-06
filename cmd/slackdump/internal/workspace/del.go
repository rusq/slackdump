package workspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
)

var (
	ErrOpCancelled = errors.New("operation cancelled")
	ErrNotExists   = errors.New("workspace does not exist")
)

var CmdWspDel = &base.Command{
	UsageLine: baseCommand + " del [flags]",
	Short:     "deletes the saved workspace credentials",
	Long: `
# Workspace Delete Command

Use ` + "`del`" + ` to delete the Slack Workspace login information ("forget"
the workspace).

If the workspace login information is deleted, you will need to login into that
workspace again by running ` + " `slackdump auth new <name>`" + `, in case you
need to use this workspace again.

Slackdump will ask for the confirmation before deleting.  To omit the
question, use ` + "`-y`" + ` flag.
`,
	CustomFlags: false,
	FlagMask:    cfg.OmitAll,
	PrintFlags:  true,
}

func init() {
	CmdWspDel.Run = runWspDel
}

var (
	delAll     = CmdWspDel.Flag.Bool("a", false, "delete all workspaces")
	delConfirm = CmdWspDel.Flag.Bool("y", false, "answer 'yes' to all questions")
)

func runWspDel(ctx context.Context, cmd *base.Command, args []string) error {
	m, err := cache.NewManager(cfg.CacheDir())
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return err
	}
	if *delAll {
		return delAllWsp(m, *delConfirm)
	} else {
		return delOneWsp(m, args)
	}
}

func delAllWsp(m manager, confirm bool) error {
	workspaces, err := m.List()
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	if !confirm && !yesno("This will delete ALL workspaces") {
		base.SetExitStatus(base.SCancelled)
		return ErrOpCancelled
	}
	for _, name := range workspaces {
		if err := m.Delete(name); err != nil {
			base.SetExitStatus(base.SCacheError)
			return err
		}
		fmt.Printf("workspace %q deleted\n", name)
	}
	return nil
}

func delOneWsp(m manager, args []string) error {
	wsp := argsWorkspace(args, cfg.Workspace)
	if wsp == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return cache.ErrNameRequired
	}

	if !m.Exists(wsp) {
		base.SetExitStatus(base.SUserError)
		return ErrNotExists
	}

	if !*delConfirm && !yesno(fmt.Sprintf("workspace %q is about to be deleted", wsp)) {
		base.SetExitStatus(base.SNoError)
		return ErrOpCancelled
	}

	if err := m.Delete(wsp); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	fmt.Printf("workspace %q deleted\n", wsp)
	return nil
}
