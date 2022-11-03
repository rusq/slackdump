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

// argsWorkspace checks if the current workspace override is set, and returns it
// if it is. Otherwise, it checks the first (with index zero) argument in args,
// and if it set, returns it.  Otherwise, it returns an empty string.
func argsWorkspace(args []string) string {
	if cfg.Workspace != "" {
		return cfg.Workspace
	}
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}

	return ""
}
