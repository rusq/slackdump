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
	"fmt"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
)

var cmdWspSelect = &base.Command{
	UsageLine: baseCommand + " select [flags]",
	Short:     "choose a previously saved workspace",
	Long: `
# Workspace Select Command

**Select** allows to set the current workspace from the list of workspaces
that you have previously authenticated in.

"Current" means that this workspace will be used by default when running
other commands, unless you specify a different workspace explicitly with
the ` + "`-w`" + ` flag.

To get the full list of authenticated workspaces, run:

	` + base.Executable() + ` auth list
`,
	FlagMask:   flagmask,
	PrintFlags: true,
	Wizard:     wizSelect,
}

func init() {
	cmdWspSelect.Run = runSelect
}

func runSelect(ctx context.Context, cmd *base.Command, args []string) error {
	wsp := argsWorkspace(args, cfg.Workspace)
	if wsp == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return cache.ErrNameRequired
	}
	m, err := CacheMgr()
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return fmt.Errorf("unable to initialise cache: %s", err)
	}
	// TODO: maybe ask the user to create new workspace if the workspace
	// does not exist.
	if err := m.Select(wsp); err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("unable to select %q: %w", args[0], err)
	}
	fmt.Printf("Success:  current workspace set to:  %s\n", args[0])
	return nil
}
