package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/trace"
	"strings"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
)

const baseCommand = "slackdump workspace"

var flagmask = cfg.OmitAll &^ cfg.OmitCacheDir

var CmdWorkspace = &base.Command{
	Run:       nil,
	Wizard:    nil,
	UsageLine: baseCommand,
	Short:     "add or choose already existing workspace to run on",
	Long: `
# Workspace Command

Slackdump supports working with multiple Slack Workspaces without the need
to authenticate again (unless login credentials are expired or became invalid
due to some other reason).

**Workspace** command allows to **add** a new Slack Workspace, **list** already 
authenticated workspaces, **select** a workspace that you have previously
logged in to, or **del**ete an existing workspace.

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
		CmdWspNew,
		CmdWspList,
		CmdWspSelect,
		CmdWspDel,
	},
}

//go:generate mockgen -destination=mocks_test.go -package=workspace -source=workspace.go manager

// manager is used for test rigging.
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
	m, err := cache.NewManager(cacheDir)
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

var yesno = base.YesNo

// authWsp authenticates in the workspace wsp, and saves, or reuses the
// credentials in the cacheDir.  It returns ErrNotExists if the workspace
// doesn't exist in the cacheDir.
func authWsp(ctx context.Context, cacheDir string, wsp string, usePlaywright bool) (auth.Provider, error) {
	m, err := cache.NewManager(cacheDir)
	if err != nil {
		return nil, err
	}
	if err := m.ExistsErr(wsp); err != nil {
		return nil, err
	}

	prov, err := m.Auth(ctx, wsp, cache.AuthData{Token: cfg.SlackToken, Cookie: cfg.SlackCookie, UsePlaywright: usePlaywright})
	if err != nil {
		return nil, err
	}
	return prov, nil
}
