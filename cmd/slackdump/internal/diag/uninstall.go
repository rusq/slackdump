package diag

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rusq/slackauth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/diag/info"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v3/logger"
)

var CmdUninstall = &base.Command{
	UsageLine:   "slackdump tools uninstall",
	Short:       "performs uninstallation of components",
	RequireAuth: false,
	FlagMask:    cfg.OmitAll,
	Run:         runUninstall,
	PrintFlags:  true,
}

func init() {
	CmdUninstall.Wizard = wizUninstall
}

type uninstOptions struct {
	legacy    bool // playwright
	dry       bool // dry run
	noConfirm bool // no confirmation from the user
}

// uninstParams holds supported command line parameters
var uninstParams = uninstOptions{}

func init() {
	CmdUninstall.Flag.BoolVar(&uninstParams.legacy, "legacy-browser", false, "operate on playwright environment (default: rod envronment)")
	CmdUninstall.Flag.BoolVar(&uninstParams.dry, "dry", false, "dry run")
	CmdUninstall.Flag.BoolVar(&uninstParams.noConfirm, "no-confirm", false, "no confirmation from the user")
}

func runUninstall(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) != 0 {
		base.SetExitStatus(base.SInvalidParameters)
	}
	if !uninstParams.noConfirm {
		confirmed, err := ui.Confirm("This will uninstall the EZ-Login browser", true)
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	si := info.CollectRaw()

	if uninstParams.legacy {
		return uninstallPlaywright(ctx, si.Playwright, uninstParams.dry)
	} else {
		return uninstallRod(ctx, si.Rod, uninstParams.dry)
	}
}

func removeFunc(dry bool) func(string) error {
	var removeFn = os.RemoveAll
	if dry {
		removeFn = func(name string) error {
			fmt.Printf("Would remove %s\n", name)
			return nil
		}
	}
	return removeFn
}

func uninstallPlaywright(ctx context.Context, si info.PwInfo, dry bool) error {
	removeFn := removeFunc(dry)
	lg := logger.FromContext(ctx)
	lg.Printf("Deleting %s", si.Path)
	if err := removeFn(si.Path); err != nil {
		return fmt.Errorf("failed to remove the playwright library: %w", err)
	}
	lg.Printf("Deleting browsers in %s", si.BrowsersPath)

	if err := removeFn(si.BrowsersPath); err != nil {
		return fmt.Errorf("failed to remove the playwright browsers: %w", err)
	}
	dir, _ := filepath.Split(si.Path)
	if len(dir) == 0 {
		return errors.New("unable to reliably determine playwright path")
	}
	lg.Printf("Deleting all playwright versions from:  %s", dir)
	if err := removeFn(dir); err != nil {
		return fmt.Errorf("failed to remove the playwright versions: %w", err)
	}

	return nil
}

func uninstallRod(_ context.Context, si info.RodInfo, dry bool) error {
	removeFn := removeFunc(dry)
	if si.Path == "" {
		return errors.New("unable to determine rod browser path")
	}
	lg := cfg.Log
	lg.Printf("Deleting incognito Browser...")
	if !dry {
		_ = slackauth.RemoveBrowser() // just to make sure.
	} else {
		lg.Printf("Would remove incognito browser")
	}

	lg.Printf("Deleting %s...", si.Path)
	if err := removeFn(si.Path); err != nil {
		return fmt.Errorf("failed to remove the rod browser: %w", err)
	}

	return nil
}

func wizUninstall(ctx context.Context, cmd *base.Command, args []string) error {
	w := dumpui.Wizard{
		Name:        "Uninstall",
		Title:       "Uninstall Slackdump",
		LocalConfig: uninstParams.configuration,
		Cmd:         CmdUninstall,
	}
	return w.Run(ctx)
}

func (p *uninstOptions) configuration() cfgui.Configuration {
	p.noConfirm = true
	return cfgui.Configuration{
		{
			Name: "Uninstall options",
			Params: []cfgui.Parameter{
				{
					Name:        "Playwright",
					Value:       cfgui.Checkbox(p.legacy),
					Description: "Environment to uninstall (if unselected, uninstalls Rod)",
					Updater:     updaters.NewBool(&p.legacy),
				},
				{
					Name:        "Dry run",
					Value:       cfgui.Checkbox(p.dry),
					Description: "Do not perform the uninstallation, just show what would be done",
					Updater:     updaters.NewBool(&p.dry),
				},
				// TODO: delete slackdump from user cache options.
			},
		},
	}
}
