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

// Package export implements export subcommand.
package export

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rusq/slackdump/v4/source"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/structures"
)

var CmdExport = &base.Command{
	Run:         nil,
	Wizard:      nil,
	UsageLine:   "slackdump export",
	Short:       "exports the Slack Workspace or individual conversations",
	FlagMask:    cfg.OmitUserCacheFlag | cfg.OmitRecordFilesFlag,
	Long:        mdExport,
	CustomFlags: false,
	PrintFlags:  true,
	RequireAuth: true,
}

//go:embed assets/export.md
var mdExport string

type exportFlags struct {
	ExportStorageType source.StorageType
	ExportToken       string
}

var options = exportFlags{
	ExportStorageType: source.STmattermost,
}

func init() {
	CmdExport.Flag.Var(&options.ExportStorageType, "type", "export file storage type")
	CmdExport.Flag.StringVar(&options.ExportToken, "export-token", "", "file export token to append to each of the file URLs")

	CmdExport.Run = runExport
	CmdExport.Wizard = wizExport
}

func runExport(ctx context.Context, cmd *base.Command, args []string) error {
	start := time.Now()
	if strings.TrimSpace(cfg.Output) == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("use -base to set the base output location")
	}
	if !cfg.WithFiles {
		options.ExportStorageType = source.STnone
	}
	list, err := structures.NewEntityList(args)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("error parsing the entity list: %w", err)
	}
	if err := bootstrap.AskOverwrite(cfg.Output); err != nil {
		return err
	}

	client, err := bootstrap.Slack(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	fsa, err := fsadapter.New(cfg.Output)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	lg := cfg.Log
	defer func() {
		lg.DebugContext(ctx, "closing the fsadapter")
		fsa.Close()
	}()

	// TODO: remove once the database is stable.
	if cfg.UseChunkFiles {
		lg.DebugContext(ctx, "using chunk files backend")
		err = exportWithDir(ctx, client, fsa, list, options)
	} else {
		lg.DebugContext(ctx, "using database backend")
		err = exportWithDB(ctx, client, fsa, list, options)
	}
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("export failed: %w", err)
	}

	lg.InfoContext(ctx, "export completed", "output", cfg.Output, "took", time.Since(start).String())
	return nil
}
