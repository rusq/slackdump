package workspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/app/appauth"
)

var ErrOpCancelled = errors.New("operation cancelled")

var CmdWspDel = &base.Command{
	UsageLine: "slackdump workspace del [flags]",
	Short:     "deletes the saved workspace login information",
	Long: `
Del can be used to delete the Slack Workspace login information (forgets the
workspace).

If the workspace login information is deleted, you will need to re-authorize
in that Slack Workspace by running "slackdump workspace new <name>".
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
	delConfirm = CmdWspDel.Flag.Bool("y", false, "answer yes to all questions")
)

func runWspDel(ctx context.Context, cmd *base.Command, args []string) error {
	if *delAll {
		return delAllWsp()
	} else {
		return delOneWsp(args)
	}
}

func delAllWsp() error {
	m, err := appauth.NewManager(cfg.CacheDir())
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return err
	}

	workspaces, err := m.List()
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	if !*delConfirm && !base.YesNo("This will delete ALL workspaces") {
		base.SetExitStatus(base.SNoError)
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

func delOneWsp(args []string) error {
	wsp := argsWorkspace(args)
	if wsp == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return appauth.ErrNameRequired
	}

	m, err := appauth.NewManager(cfg.CacheDir())
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return err
	}

	if !m.Exists(wsp) {
		base.SetExitStatus(base.SUserError)
		return errors.New("workspace does not exist")
	}

	if !*delConfirm && !base.YesNo(fmt.Sprintf("workspace %q is about to be deleted", wsp)) {
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
