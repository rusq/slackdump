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
	Long: base.Render(`
# Workspace Command

Slackdump supports working with multiple Slack Workspaces without the need
to authenticate again (unless login credentials are expired or became invalid
due to some other reason).

**Workspace** command allows to authenticate in a **new** Slack Workspace,
**list** already authenticated workspaces, **select** a workspace that you have
previously logged in to, or **del**ete an existing workspace.

To learn more about different login options, run:

	slackdump help login

Workspaces are stored on this device in the Cache directory, which is
automatically detected to be:
    ` + cfg.CacheDir() + `
`),
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
