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

package workspace

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/cache"
)

var (
	ErrOpCancelled = errors.New("operation cancelled")
	ErrNotExists   = errors.New("workspace does not exist")
)

//go:embed assets/del.md
var delMD string

var cmdWspDel = &base.Command{
	UsageLine:   baseCommand + " del [flags]",
	Short:       "deletes the saved workspace credentials",
	Long:        delMD,
	CustomFlags: false,
	FlagMask:    cfg.OmitAll,
	PrintFlags:  true,
}

func init() {
	cmdWspDel.Run = runWspDel
}

var (
	delAll = cmdWspDel.Flag.Bool("a", false, "delete all workspaces")
)

func runWspDel(ctx context.Context, cmd *base.Command, args []string) error {
	m, err := CacheMgr()
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return err
	}
	if *delAll {
		return delAllWsp(m, cfg.YesMan)
	} else {
		return delOneWsp(m, args)
	}
}

func delAllWsp(m manager, confirm bool) error {
	workspaces, err := m.List()
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	if !confirm && !yesno("This will delete ALL workspaces") {
		base.SetExitStatus(base.SCancelled)
		return base.ErrOpCancelled
	}
	for _, name := range workspaces {
		if err := m.Delete(name); err != nil {
			base.SetExitStatus(base.SCacheError)
			return err
		}
		fmt.Printf("workspace %q deleted\n", name)
	}
	return nil
}

func delOneWsp(m manager, args []string) error {
	wsp := argsWorkspace(args, cfg.Workspace)
	if wsp == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return cache.ErrNameRequired
	}

	if !m.Exists(wsp) {
		base.SetExitStatus(base.SUserError)
		return ErrNotExists
	}

	if !cfg.YesMan && !yesno(fmt.Sprintf("workspace %q is about to be deleted", wsp)) {
		base.SetExitStatus(base.SNoError)
		return base.ErrOpCancelled
	}

	if err := m.Delete(wsp); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	fmt.Printf("workspace %q deleted\n", wsp)
	return nil
}
