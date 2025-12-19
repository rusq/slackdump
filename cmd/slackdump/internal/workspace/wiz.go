package workspace

import (
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace/workspaceui"
)

var cmdWspWiz = &base.Command{
	Run:       workspaceui.WorkspaceNew,
	UsageLine: baseCommand + " wiz [flags]",
	Short:     "starts new workspace wizard",
	Long: `# Workspace Wizard
Use this command to start the Workspace creation wizard.

The behaviour is the same as if one chooses Slackdump Wizard ==> Workspace ==> New.
`,
	FlagMask:    cfg.OmitAll,
	PrintFlags:  false,
	RequireAuth: false,
	HideWizard:  true,
}
