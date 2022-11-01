package workspace

import (
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var flagmask = cfg.OmitAll

var CmdWorkspace = &base.Command{
	Run:       nil,
	Wizard:    nil,
	UsageLine: "slackdump workspace",
	Short:     "authenticate or choose workspace to run on",
	Long: `
Slackdump supports working with multiple Slack Workspaces without the need
to authenticate again (unless login credentials are expired).

Workspace command allows to authenticate in a new Slack Workspace, list already
authenticated workspaces, and choose a workspace that you have previously
authenticated.

Run:

	slackdump help login

To learn more about different login options.

Workspaces are stored in cache directory on this device:
` + cfg.CacheDir() + `
`,
	CustomFlags: false,
	FlagMask:    flagmask,
	PrintFlags:  false,
	RequireAuth: false,
	Commands: []*base.Command{
		CmdWspNew,
		CmdWspList,
		CmdWspSelect,
		CmdWspDel,
	},
}
