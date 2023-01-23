package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/trace"

	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/cache"
	cache2 "github.com/rusq/slackdump/v2/internal/cache"
)

var flagmask = cfg.OmitAll

var CmdWorkspace = &base.Command{
	Run:       nil,
	Wizard:    nil,
	UsageLine: "slackdump workspace",
	Short:     "authenticate or choose workspace to run on",
	Long: `
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

// manager is used for test rigging.
type manager interface {
	Auth(ctx context.Context, name string, c cache.Credentials) (auth.Provider, error)
	Delete(name string) error
	Exists(name string) bool
	FileInfo(name string) (os.FileInfo, error)
	List() ([]string, error)
}

// argsWorkspace checks if the current workspace override is set, and returns it
// if it is. Otherwise, it checks the first (with index zero) argument in args,
// and if it set, returns it.  Otherwise, it returns an empty string.
func argsWorkspace(args []string, defaultWsp string) string {
	if defaultWsp != "" {
		return defaultWsp
	}
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}

	return ""
}

// Auth authenticates in the workspace wsp, and saves, or reuses the credentials
// in the dir.
func Auth(ctx context.Context, dir string, wsp string) (auth.Provider, error) {
	m, err := cache2.NewManager(dir)
	if err != nil {
		return nil, err
	}
	if !m.Exists(wsp) {
		return nil, fmt.Errorf("workspace does not exist: %q", cfg.Workspace)
	}

	prov, err := m.Auth(ctx, wsp, cache2.SlackCreds{Token: cfg.SlackToken, Cookie: cfg.SlackCookie})
	if err != nil {
		return nil, err
	}
	return prov, nil
}

// AuthCurrent authenticates in the current workspace, or overrideWsp if it's
// provided.
func AuthCurrent(ctx context.Context, cacheDir string, overrideWsp string) (auth.Provider, error) {
	wsp, err := Current(cacheDir, overrideWsp)
	if err != nil {
		return nil, err
	}
	trace.Logf(ctx, "AuthCurrent", "current workspace=%s", wsp)

	prov, err := Auth(ctx, cacheDir, wsp)
	if err != nil {
		return nil, err
	}
	return prov, nil
}

// AuthCurrentCtx authenticates in the current or overriden workspace and
// returns the context with the auth.Provider.
func AuthCurrentCtx(pctx context.Context, cacheDir string, overrideWsp string) (context.Context, error) {
	prov, err := AuthCurrent(pctx, cacheDir, overrideWsp)
	if err != nil {
		return nil, err
	}
	return auth.WithContext(pctx, prov), nil
}

// Current returns the current workspace in the directory dir, based on the
// configuration values.  If cfg.Workspace is set, it checks if the workspace
// cfg.Workspace exists in the directory dir, and returns it.
func Current(dir string, override string) (wsp string, err error) {
	m, err := cache2.NewManager(dir)
	if err != nil {
		return "", err
	}

	if override != "" {
		if m.Exists(override) {
			return override, nil
		}
		return "", fmt.Errorf("workspace does not exist: %q", override)
	}

	wsp, err = m.Current()
	if err != nil {
		if errors.Is(err, cache2.ErrNoWorkspaces) {
			wsp = "default"
		} else {
			return "", err
		}
	}
	return wsp, nil
}
