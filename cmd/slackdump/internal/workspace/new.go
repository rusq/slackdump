package workspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
)

var CmdWspNew = &base.Command{
	UsageLine: baseCommand + " new [flags] name",
	Short:     "authenticate in a Slack Workspace",
	Long: `
# Auth New Command

**New** allows you to authenticate in an existing Slack Workspace.
`,
	FlagMask:   flagmask &^ cfg.OmitAuthFlags, // only auth flags.
	PrintFlags: true,
}

var newParams = struct {
	confirm bool
}{}

func init() {
	CmdWspNew.Flag.BoolVar(&newParams.confirm, "y", false, "answer yes to all questions")

	CmdWspNew.Run = runWspNew
}

// runWspNew authenticates in the new workspace.
func runWspNew(ctx context.Context, cmd *base.Command, args []string) error {
	lg := cfg.Log
	m, err := cache.NewManager(
		cfg.CacheDir(),
		cache.WithAuthOpts(
			auth.BrowserWithBrowser(cfg.Browser),
			auth.BrowserWithTimeout(cfg.LoginTimeout),
			auth.RODWithRODHeadlessTimeout(cfg.HeadlessTimeout),
			auth.RODWithUserAgent(cfg.RODUserAgent),
		))
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return fmt.Errorf("error initialising workspace manager: %s", err)
	}

	wsp := argsWorkspace(args, cfg.Workspace)

	if m.Exists(realname(wsp)) {
		if !newParams.confirm && !base.YesNo(fmt.Sprintf("Workspace %q already exists. Overwrite", realname(wsp))) {
			return ErrOpCancelled
		}
		if err := m.Delete(realname(wsp)); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
	}

	lg.Debugln("requesting authentication...")
	ad := cache.AuthData{
		Token:         cfg.SlackToken,
		Cookie:        cfg.SlackCookie,
		UsePlaywright: cfg.LegacyBrowser,
	}
	prov, err := m.Auth(ctx, wsp, ad)
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
