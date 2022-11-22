package workspace

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	cache2 "github.com/rusq/slackdump/v2/internal/cache"
)

var CmdWspNew = &base.Command{
	UsageLine: "slackdump workspace new [flags] name",
	Short:     "authenticate in a Slack Workspace",
	Long: base.Render(`
# Workspace New Command

**New** allows you to authenticate in an existing Slack Workspace.
`),
	FlagMask:   flagmask &^ cfg.OmitAuthFlags,
	PrintFlags: true,
}

var (
	newConfirm = CmdWspNew.Flag.Bool("y", false, "answer yes to all questions")
)

func init() {
	CmdWspNew.Run = runWspNew
}

// runWspNew authenticates in the new workspace.
func runWspNew(ctx context.Context, cmd *base.Command, args []string) error {
	m, err := cache2.NewManager(cfg.CacheDir(), cache2.WithAuthOpts(auth.BrowserWithBrowser(cfg.Browser)))
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return fmt.Errorf("error initialising workspace manager: %s", err)
	}

	wsp := argsWorkspace(args, cfg.Workspace)

	if m.Exists(realname(wsp)) {
		if !*newConfirm && !base.YesNo(fmt.Sprintf("Workspace %q already exists. Overwrite", realname(wsp))) {
			return ErrOpCancelled
		}
		if err := m.Delete(realname(wsp)); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
	}

	creds := cache2.SlackCreds{
		Token:  cfg.SlackToken,
		Cookie: cfg.SlackCookie,
	}
	prov, err := m.Auth(ctx, wsp, creds)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}

	// select it
	if err := m.Select(realname(wsp)); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("failed to select the default workpace: %s", err)
	}
	fmt.Printf("Success:  added workspace %q of type %q\n", realname(wsp), prov.Type())
	return nil
}

func realname(name string) string {
	if name == "" {
		return "default"
	}
	return name
}
