package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/app/appauth"
)

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

func runWspDel(ctx context.Context, cmd *base.Command, args []string) {
	if *delAll {
		delAllWsp()
	} else {
		delOneWsp(args)
	}
}

func delAllWsp() {
	m, err := appauth.NewManager(cfg.CacheDir())
	if err != nil {
		base.SetExitStatusMsg(base.SCacheError, err.Error())
		return
	}

	workspaces, err := m.List()
	if err != nil {
		base.SetExitStatusMsg(base.SApplicationError, err.Error())
	}

	if !*delConfirm && !yesno("This will delete ALL workspaces") {
		base.SetExitStatusMsg(base.SNoError, "operation cancelled")
		return
	}
	for _, name := range workspaces {
		if err := m.Delete(name); err != nil {
			base.SetExitStatusMsg(base.SCacheError, err.Error())
			return
		}
		fmt.Printf("workspace %q deleted\n", name)
	}
}

func delOneWsp(args []string) {
	wsp, err := argsWorkspace(args)
	if err != nil {
		base.SetExitStatusMsg(base.SInvalidParameters, err.Error())
		return
	}

	m, err := appauth.NewManager(cfg.CacheDir())
	if err != nil {
		base.SetExitStatusMsg(base.SCacheError, err.Error())
		return
	}

	if !m.Exists(wsp) {
		base.SetExitStatusMsg(base.SUserError, "workspace does not exist")
		return
	}

	if !*delConfirm && !yesno(fmt.Sprintf("workspace %q is about to be deleted", wsp)) {
		base.SetExitStatusMsg(base.SNoError, "operation cancelled")
		return
	}

	if err := m.Delete(wsp); err != nil {
		base.SetExitStatusMsg(base.SApplicationError, err.Error())
		return
	}
	fmt.Printf("workspace %q deleted\n", wsp)
}

func yesno(message string) bool {
	for {
		fmt.Print(message, "? (y/N) ")
		var resp string
		fmt.Scanln(&resp)
		resp = strings.TrimSpace(resp)
		if len(resp) > 0 {
			switch strings.ToLower(resp)[0] {
			case 'y':
				return true
			case 'n':
				return false
			}
		}
		fmt.Println("Please answer yes or no and press Enter or Return.")
	}
}
