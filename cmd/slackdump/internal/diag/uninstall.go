// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package diag

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rusq/slackauth"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/diag/info"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/cfgui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/dumpui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/updaters"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace"
	"github.com/rusq/slackdump/v3/internal/cache"
)

var cmdUninstall = &base.Command{
	UsageLine:   "slackdump tools uninstall",
	Short:       "performs uninstallation of components",
	RequireAuth: false,
	FlagMask:    cfg.OmitAll,
	Run:         runUninstall,
	PrintFlags:  true,
}

func init() {
	cmdUninstall.Wizard = wizUninstall
}

type uninstOptions struct {
	playwright bool // remove playwright
	rod        bool // remove rod
	cache      bool // remove user cache
	purge      bool // remove everything

	dry       bool // dry run
	noConfirm bool // no confirmation from the user
}

func (o uninstOptions) selected() []string {
	const (
		rod   = "Rod Browser"
		pw    = "Playwright Browsers"
		cache = "User Cache"
	)
	items := []string{}
	if o.purge {
		return []string{rod, pw, cache}
	}
	if o.rod {
		items = append(items, rod)
	}
	if o.playwright {
		items = append(items, pw)
	}
	if o.cache {
		items = append(items, cache)
	}
	return items
}

func (o uninstOptions) String() string {
	var buf strings.Builder
	for _, s := range o.selected() {
		buf.WriteString("* " + s)
		buf.WriteString("\n")
	}
	return buf.String()
}

// uninstParams holds supported command line parameters
var uninstParams = uninstOptions{}

func init() {
	cmdUninstall.Flag.BoolVar(&uninstParams.playwright, "legacy-browser", false, "alias for -playwright")
	cmdUninstall.Flag.BoolVar(&uninstParams.playwright, "playwright", false, "remove playwright environment")
	cmdUninstall.Flag.BoolVar(&uninstParams.rod, "browser", false, "remove rod browser")
	cmdUninstall.Flag.BoolVar(&uninstParams.cache, "cache", false, "remove saved workspaces and user/channel cache")
	cmdUninstall.Flag.BoolVar(&uninstParams.purge, "purge", false, "remove everything (same as -rod -playwright -cache)")
	cmdUninstall.Flag.BoolVar(&uninstParams.dry, "dry", false, "dry run")
	cmdUninstall.Flag.BoolVar(&uninstParams.noConfirm, "no-confirm", false, "no confirmation from the user")
}

func runUninstall(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) != 0 {
		base.SetExitStatus(base.SInvalidParameters)
	}
	if len(uninstParams.selected()) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("nothing to uninstall")
	}

	m, err := workspace.CacheMgr()
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return err
	}
	if !uninstParams.noConfirm {
		confirmed, err := ui.Confirm(fmt.Sprintf("This will uninstall the following:\n\n%s", uninstParams), true)
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	si := info.CollectRaw()

	if uninstParams.purge {
		uninstParams.cache = true
		uninstParams.playwright = true
		uninstParams.rod = true
	}

	if uninstParams.cache {
		if err := removeCache(m, uninstParams.dry); err != nil {
			base.SetExitStatus(base.SCacheError)
			return err
		}
	}
	if uninstParams.playwright {
		if err := uninstallPlaywright(ctx, si.Playwright, uninstParams.dry); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
	}
	if uninstParams.rod {
		if err := uninstallRod(ctx, si.Rod, uninstParams.dry); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
	}
	return nil
}

func removeFunc(dry bool) func(string) error {
	if !dry {
		return os.RemoveAll
	}
	return func(name string) error {
		fmt.Printf("Would remove %s\n", name)
		return nil
	}
}

func uninstallPlaywright(ctx context.Context, si info.PwInfo, dry bool) error {
	removeFn := removeFunc(dry)
	lg := cfg.Log.WithGroup("playwright")
	lg.InfoContext(ctx, "Deleting", "path", si.Path)
	if err := removeFn(si.Path); err != nil {
		return fmt.Errorf("failed to remove the playwright library: %w", err)
	}

	lg.InfoContext(ctx, "Deleting browsers", "browsers_path", si.BrowsersPath)
	if err := removeFn(si.BrowsersPath); err != nil {
		return fmt.Errorf("failed to remove the playwright browsers: %w", err)
	}
	dir, _ := filepath.Split(si.Path)
	if len(dir) == 0 {
		return errors.New("unable to reliably determine playwright path")
	}
	lg.InfoContext(ctx, "Deleting all playwright versions", "dir", dir)
	if err := removeFn(dir); err != nil {
		return fmt.Errorf("failed to remove the playwright versions: %w", err)
	}

	return nil
}

func uninstallRod(ctx context.Context, si info.RodInfo, dry bool) error {
	lg := cfg.Log.WithGroup("rod")

	removeFn := removeFunc(dry)
	if si.Path == "" {
		return errors.New("unable to determine rod browser path")
	}
	lg.InfoContext(ctx, "Deleting ROD Browser...")
	if !dry {
		_ = slackauth.RemoveBrowser() // just to make sure.
	} else {
		lg.InfoContext(ctx, "Would remove incognito browser")
	}

	lg.InfoContext(ctx, "Deleting...", "path", si.Path)
	if err := removeFn(si.Path); err != nil {
		return fmt.Errorf("failed to remove the rod browser: %w", err)
	}

	return nil
}

func wizUninstall(ctx context.Context, cmd *base.Command, args []string) error {
	w := dumpui.Wizard{
		Name:        "Uninstall",
		Title:       "Uninstall Slackdump Components",
		LocalConfig: uninstParams.configuration,
		Cmd:         cmdUninstall,
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
					Value:       cfgui.Checkbox(p.playwright),
					Description: "Environment to uninstall (if unselected, uninstalls Rod)",
					Updater:     updaters.NewBool(&p.playwright),
				},
				{
					Name:        "Rod Browser",
					Value:       cfgui.Checkbox(p.rod),
					Description: "Browser to uninstall (if unselected, uninstalls Playwright)",
					Updater:     updaters.NewBool(&p.rod),
				},
				{
					Name:        "User cache",
					Value:       cfgui.Checkbox(p.cache),
					Description: "Remove all saved workspaces and user/channel cache",
					Updater:     updaters.NewBool(&p.cache),
				},
				{
					Name:        "Dry run",
					Value:       cfgui.Checkbox(p.dry),
					Description: "Do not perform the uninstallation, just show what would be done",
					Updater:     updaters.NewBool(&p.dry),
				},
			},
		},
	}
}

func removeCache(m *cache.Manager, dry bool) error {
	lg := cfg.Log.WithGroup("cache")
	lg.Info("Removing cache at ", "path", cfg.CacheDir())
	if dry {
		fmt.Println("Would remove cache")
		return nil
	}
	if err := m.RemoveAll(); err != nil {
		return fmt.Errorf("failed to remove cache: %w", err)
	}
	return nil
}
