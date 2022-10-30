package workspace

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/app/appauth"
)

var CmdWspNew = &base.Command{
	Run:       runWspNew,
	UsageLine: "slackdump workspace new [flags] name",
	Short:     "authenticate in a Slack Workspace",
	Long: `
New allows you to authenticate in an existing Slack Workspace.
`,
	FlagMask:   flagmask,
	PrintFlags: true,
}

func runWspNew(ctx context.Context, cmd *base.Command, args []string) {
	if cfg.Workspace == "" {
		if args[0] == "" {
			base.SetExitStatusMsg(base.SInvalidParameters, "workspace name must be specified")
			return
		}
		cfg.Workspace = args[0]
	}

	m, err := appauth.NewManager(cfg.CacheDir())
	if err != nil {
		base.SetExitStatusMsg(base.SCacheError, fmt.Sprintf("error initialising workspace manager: %s", err))
		return
	}
	creds := appauth.SlackCreds{
		Token:  cfg.SlackToken,
		Cookie: cfg.SlackCookie,
	}
	prov, err := m.Auth(ctx, cfg.Workspace, creds)
	if err != nil {
		base.SetExitStatusMsg(base.SAuthError, err)
		return
	}
	fmt.Printf("Success:  added workspace %q of type %q\n", cfg.Workspace, prov.Type())
}
