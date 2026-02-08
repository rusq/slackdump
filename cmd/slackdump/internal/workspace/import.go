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

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
)

//go:embed assets/import.md
var importMd string

var cmdWspImport = &base.Command{
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
	m, err := CacheMgr()
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
