package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/trace"
	"strings"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace/workspaceui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace/wspcfg"
	"github.com/rusq/slackdump/v3/internal/cache"
)

const baseCommand = "slackdump workspace"

var flagmask = cfg.OmitAll &^ cfg.OmitCacheDir

var CmdWorkspace = &base.Command{
	Run:       nil,
	Wizard:    nil,
	UsageLine: baseCommand,
	Short:     "manage Slack Workspaces",
	Long: `
# Workspace Command

Slackdump supports working with multiple Slack Workspaces without the need
to authenticate again (unless login credentials are expired or became invalid
due to some other reason).

**Workspace** command allows to add a **new** Slack Workspace, **list** already
authenticated workspaces, **select** a workspace that you have previously
logged in to, **del**ete an existing workspace, or **import** credentials from
an environment file.

To learn more about different login options, run:

	slackdump help workspace

Workspaces are stored on this device in the system Cache directory, which is
automatically detected to be:
    ` + cfg.CacheDir() + `
`,
	CustomFlags: false,
	FlagMask:    flagmask,
	PrintFlags:  false,
	RequireAuth: false,
	Commands: []*base.Command{
		cmdWspNew,
		cmdWspImport,
		cmdWspList,
		cmdWspSelect,
		cmdWspDel,
		cmdWspWiz,
	},
}

// manager is used for test rigging.
//
//go:generate mockgen -destination=mocks_test.go -package=workspace -source=workspace.go manager
type manager interface {
	Auth(ctx context.Context, name string, c cache.Credentials) (auth.Provider, error)
	Delete(name string) error
	Exists(name string) bool
	FileInfo(name string) (os.FileInfo, error)
	List() ([]string, error)
	LoadProvider(name string) (auth.Provider, error)
	Select(name string) error
	Current() (string, error)
}

// argsWorkspace checks if the current workspace override is set, and returns it
// if it is. Otherwise, it checks the first (with index zero) argument in args,
// and if it set, returns it.  Otherwise, it returns an empty string.
func argsWorkspace(args []string, defaultWsp string) string {
	if strings.TrimSpace(defaultWsp) != "" {
		return strings.ToLower(defaultWsp)
	}
	if len(args) > 0 && args[0] != "" {
		return strings.ToLower(args[0])
	}

	return ""
}

// AuthCurrent authenticates in the current workspace, or overrideWsp if it's
// provided.
func AuthCurrent(ctx context.Context, cacheDir string, overrideWsp string, usePlaywright bool) (auth.Provider, error) {
	wsp, err := Current(cacheDir, overrideWsp)
	if err != nil {
		return nil, err
	}
	trace.Logf(ctx, "AuthCurrent", "current workspace=%s", wsp)
	slog.DebugContext(ctx, "current", "workspace", wsp)

	prov, err := authWsp(ctx, cacheDir, wsp, usePlaywright)
	if err != nil {
		return nil, err
	}
	return prov, nil
}

// Current returns the current workspace in the directory dir, based on the
// configuration values.  If cfg.Workspace is set, it checks if the workspace
// cfg.Workspace exists in the directory dir, and returns it.
func Current(cacheDir string, override string) (wsp string, err error) {
	m, err := cache.NewManager(cacheDir, mgrOpts()...)
	if err != nil {
		return "", err
	}
	if override != "" {
		if m.Exists(override) {
			return override, nil
		}
		return "", fmt.Errorf("%w: %q", ErrNotExists, override)
	}

	wsp, err = m.Current()
	if err != nil {
		if errors.Is(err, cache.ErrNoWorkspaces) {
			wsp = "default"
		} else {
			return "", err
		}
	}
	return wsp, nil
}

func CurrentName() string {
	if current, err := Current(cfg.CacheDir(), cfg.Workspace); err == nil {
		return current
	}
	return "<not set>"
}

var yesno = base.YesNo

// authWsp authenticates in the workspace wsp, and saves, or reuses the
// credentials in the cacheDir.  It returns ErrNotExists if the workspace
// doesn't exist in the cacheDir.
func authWsp(ctx context.Context, cacheDir string, wsp string, usePlaywright bool) (auth.Provider, error) {
	m, err := cache.NewManager(cacheDir, mgrOpts()...)
	if err != nil {
		return nil, err
	}
	if err := m.ExistsErr(wsp); err != nil {
		return nil, err
	}

	prov, err := m.Auth(ctx, wsp, cache.AuthData{Token: wspcfg.SlackToken, Cookie: wspcfg.SlackCookie, UsePlaywright: usePlaywright})
	if err != nil {
		return nil, err
	}
	return prov, nil
}

func mgrOpts() []cache.Option {
	return []cache.Option{cache.WithMachineID(cfg.MachineIDOvr), cache.WithNoEncryption(cfg.NoEncryption)}
}

func CacheMgr(opts ...cache.Option) (*cache.Manager, error) {
	opts = append(mgrOpts(), opts...)
	return cache.NewManager(cfg.CacheDir(), opts...)
}

// exported for testing
var (
	authCurrent = AuthCurrent
	showUI      = workspaceui.ShowUI
)

func CurrentOrNewProviderCtx(ctx context.Context) (context.Context, error) {
	cachedir := cfg.CacheDir()
	prov, err := authCurrent(ctx, cachedir, cfg.Workspace, wspcfg.LegacyBrowser)
	if err != nil {
		if errors.Is(err, cache.ErrNoWorkspaces) {
			// ask to create a new workspace
			if err := showUI(ctx, workspaceui.WithQuickLogin(), workspaceui.WithTitle("No workspaces, please choose a login method")); err != nil {
				return ctx, fmt.Errorf("auth error: %w", err)
			}
			// one more time...
			prov, err = authCurrent(ctx, cachedir, cfg.Workspace, wspcfg.LegacyBrowser)
			if err != nil {
				return ctx, err
			}
		} else {
			return ctx, err
		}
	}
	return auth.WithContext(ctx, prov), nil
}
