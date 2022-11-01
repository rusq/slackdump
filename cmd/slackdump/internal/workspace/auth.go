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
	wsp, err := argsWorkspace(args)
	if err != nil {
		base.SetExitStatusMsg(base.SInvalidParameters, err.Error())
		return
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
	prov, err := m.Auth(ctx, wsp, creds)
	if err != nil {
		base.SetExitStatusMsg(base.SAuthError, err)
		return
	}
	fmt.Printf("Success:  added workspace %q of type %q\n", wsp, prov.Type())
}

func argsWorkspace(args []string) (string, error) {
	if cfg.Workspace != "" {
		return cfg.Workspace, nil
	}
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}
	return "", appauth.ErrNameRequired
}
