package workspace

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace/workspaceui"
	"github.com/rusq/slackdump/v3/internal/cache"
)

//go:embed assets/new.md
var newMD string

var CmdWspNew = &base.Command{
	UsageLine:  baseCommand + " new [flags] <name>",
	Short:      "authenticate in a Slack Workspace",
	Long:       newMD,
	FlagMask:   flagmask &^ cfg.OmitAuthFlags, // only auth and machine-id override flags.
	PrintFlags: true,
	Wizard:     workspaceui.WorkspaceNew,
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
	m, err := CacheMgr(
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

	if err := createWsp(ctx, m, wsp, newParams.confirm); err != nil {
		return err
	}
	return nil
}

// canOverwrite defined as a variable for testing purposes.
var canOverwrite = func(wsp string) bool {
	return yesno(fmt.Sprintf("Workspace %q already exists. Overwrite", realname(wsp)))
}

// createWsp creates a new workspace interactively.
func createWsp(ctx context.Context, m manager, wsp string, confirm bool) error {
	lg := cfg.Log
	if m.Exists(realname(wsp)) {
		if !confirm && !canOverwrite(wsp) {
			base.SetExitStatus(base.SCancelled)
			return ErrOpCancelled
		}
		if err := m.Delete(realname(wsp)); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
	}

	lg.DebugContext(ctx, "requesting authentication...")
	ad := cache.AuthData{
		Token:         cfg.SlackToken,
		Cookie:        cfg.SlackCookie,
		UsePlaywright: cfg.LegacyBrowser,
	}
	prov, err := m.Auth(ctx, wsp, ad)
	if err != nil {
		if errors.Is(err, auth.ErrCancelled) {
			base.SetExitStatus(base.SCancelled)
			lg.WarnContext(ctx, auth.ErrCancelled.Error())
			return ErrOpCancelled
		}
		base.SetExitStatus(base.SAuthError)
		return err
	}

	lg.Debug("selecting as current...", "workspace", realname(wsp))
	// select it
	if err := m.Select(realname(wsp)); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("failed to select the default workpace: %s", err)
	}
	fmt.Fprintf(os.Stdout, "Success:  added workspace %q\n", realname(wsp))
	lg.DebugContext(ctx, "workspace type", "workspace", realname(wsp), "type", fmt.Sprintf("%T", prov))
	return nil
}

// realname returns the sanitised name of the workspace, replacing the empty
// string with "default".  Empty workspace name is possible if the user
// hasn't specified the workspace name on the command line.
func realname(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	return name
}
