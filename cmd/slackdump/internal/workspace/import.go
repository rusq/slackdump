package workspace

import (
	"context"
	_ "embed"
	"errors"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
)

//go:embed assets/import.md
var importMd string

var CmdImport = &base.Command{
	UsageLine:   baseCommand + " import [flags] filename",
	Short:       "import credentials from .env or secrets.txt file",
	Long:        importMd,
	FlagMask:    flagmask,
	PrintFlags:  true,
	Run:         cmdRunImport,
	RequireAuth: false,
}

func cmdRunImport(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) != 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("missing filename")
	}

	filename := args[0]

	if err := importFile(ctx, filename); err != nil {
		return err
	}
	return nil
}

func importFile(ctx context.Context, filename string) error {
	token, cookies, err := auth.ParseDotEnv(filename)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	m, err := cache.NewManager(cfg.CacheDir())
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return err
	}
	prov, err := auth.NewValueAuth(token, cookies)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}
	wsp, err := m.CreateAndSelect(ctx, prov)
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return err
	}
	cfg.Log.InfoContext(ctx, "Workspace added and selected", "workspace", wsp)
	cfg.Log.InfoContext(ctx, "It is advised that you delete the file", "filename", filename)

	return nil
}
