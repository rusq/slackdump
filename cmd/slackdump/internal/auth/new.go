package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
	"github.com/rusq/slackdump/v3/logger"
)

var CmdWspNew = &base.Command{
	UsageLine: baseCommand + " new [flags] name",
	Short:     "authenticate in a Slack Workspace",
	Long: `
# Auth New Command

**New** allows you to authenticate in an existing Slack Workspace.
`,
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
	lg := logger.FromContext(ctx)
	m, err := cache.NewManager(cfg.CacheDir(), cache.WithAuthOpts(auth.BrowserWithBrowser(cfg.Browser), auth.BrowserWithTimeout(cfg.LoginTimeout)))
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

	lg.Debugln("requesting authentication...")
	creds := cache.SlackCreds{
		Token:  cfg.SlackToken,
		Cookie: cfg.SlackCookie,
	}
	prov, err := m.Auth(ctx, wsp, creds)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		if errors.Is(err, auth.ErrCancelled) {
			lg.Println(auth.ErrCancelled)
			return nil
		}
		return err
	}

	lg.Debugf("selecting %q as current...", realname(wsp))
	// select it
	if err := m.Select(realname(wsp)); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("failed to select the default workpace: %s", err)
	}
	fmt.Printf("Success:  added workspace %q\n", realname(wsp))
	lg.Debugf("workspace %q, type %T", realname(wsp), prov)
	return nil
}

func realname(name string) string {
	if name == "" {
		return "default"
	}
	return name
}
