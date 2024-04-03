package diag

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/diag/info"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
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

// uninstallParams holds supported command line parameters
var uninstallParams = struct {
	legacy    bool // playwright
	dry       bool // dry run
	noConfirm bool // no confirmation from the user
}{}

func init() {
	CmdUninstall.Flag.BoolVar(&uninstallParams.legacy, "legacy-browser", false, "legacy mode")
	CmdUninstall.Flag.BoolVar(&uninstallParams.dry, "dry", false, "dry run")
	CmdUninstall.Flag.BoolVar(&uninstallParams.noConfirm, "no-confirm", false, "no confirmation from the user")
}

func runUninstall(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) != 0 {
		base.SetExitStatus(base.SInvalidParameters)
	}
	if uninstallParams.dry {
		return nil
	}
	if !uninstallParams.noConfirm {
		confirmed, err := ui.Confirm("This will reinstall the EZ-Login browser", true)
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	si := info.CollectRaw()

	if uninstallParams.legacy {
		return uninstallPlaywright(ctx, si.Playwright)
	} else {
		return uninstallRod(ctx, si.Rod)
	}
}

func uninstallPlaywright(ctx context.Context, si info.PwInfo) error {
	if si.Path == "" {
		return errors.New("unable to determine playwright path")
	}
	lg := logger.FromContext(ctx)
	lg.Printf("Deleting %s", si.Path)
	if err := os.RemoveAll(si.Path); err != nil {
		return fmt.Errorf("failed to remove the playwright library: %w", err)
	}
	lg.Printf("Deleting browsers")
	if err := os.RemoveAll(si.BrowsersPath); err != nil {
		return fmt.Errorf("failed to remove the playwright browsers: %w", err)
	}
	return nil
}

func uninstallRod(ctx context.Context, si info.RodInfo) error {
	if si.Path == "" {
		return errors.New("unable to determine rod browser path")
	}
	lg := logger.FromContext(ctx)
	lg.Printf("Deleting %s", si.Path)
	if err := os.RemoveAll(si.Path); err != nil {
		return fmt.Errorf("failed to remove the rod browser: %w", err)
	}

	return nil
}
