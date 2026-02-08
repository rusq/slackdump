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
package convertcmd

import (
	"context"
	_ "embed"
	"errors"
	"io/fs"
	"os"
	"time"

	"github.com/rusq/slackdump/v3/source"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

//go:embed assets/convert.md
var convertMd string

var CmdConvert = &base.Command{
	Run:         runConvert,
	UsageLine:   "slackdump convert [flags] <source>",
	Short:       "convert slackdump chunks to various formats",
	Long:        convertMd,
	CustomFlags: false,
	FlagMask:    cfg.OmitAll & ^cfg.OmitWithFilesFlag &^ cfg.OmitOutputFlag &^ cfg.OmitWithAvatarsFlag,
	PrintFlags:  true,
}

var (
	// ErrFormat is returned when the target format is not supported.
	ErrFormat = errors.New("unsupported target format")
	// ErrSource is returned when the source type is not supported for the chosen target type.
	ErrSource = errors.New("unsupported source type")
	// ErrStorage is returned when the storage type is not supported.
	ErrStorage = errors.New("unsupported storage type")
)

type tparams struct {
	storageType source.StorageType
	sessionID   int64
}

type convertFunc func(ctx context.Context, input, output string, cflg convertflags) error

var converters = map[datafmt]convertFunc{
	Fdump:     toDump,
	Fexport:   toExport,
	Fchunk:    toChunk,
	Fdatabase: toDatabase,
}

type convertflags struct {
	includeFiles   bool
	includeAvatars bool
	outStorageType source.StorageType
	sessionID      int64 // sessionID for database->chunk conversion
	outputfmt      datafmt
}

var params = convertflags{
	outStorageType: source.STmattermost,
	sessionID:      1,
	outputfmt:      Fexport,
}

func init() {
	CmdConvert.Flag.Var(&params.outStorageType, "storage", "storage type")
	CmdConvert.Flag.Var(&params.outputfmt, "format", "output `format`")
	CmdConvert.Flag.Var(&params.outputfmt, "f", "shorthand for -format")
	CmdConvert.Flag.Int64Var(&params.sessionID, "session", params.sessionID, "session `id` for database->chunk conversion")
}

func runConvert(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) < 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("source and destination are required")
	}
	if params.outputfmt == Fdatabase && params.sessionID <= 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("session id is required for database conversion")
	}
	fn, exist := converters[params.outputfmt]
	if !exist {
		base.SetExitStatus(base.SInvalidParameters)
		return ErrFormat
	}
	lg := cfg.Log
	lg.InfoContext(ctx, "converting", "source", args[0], "output_format", params.outputfmt)

	if err := bootstrap.AskOverwrite(cfg.Output); err != nil {
		return err
	}

	// set from the global config
	params.includeFiles = cfg.WithFiles
	params.includeAvatars = cfg.WithAvatars

	start := time.Now()
	if err := fn(ctx, args[0], cfg.Output, params); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	lg.InfoContext(ctx, "completed", "took", time.Since(start))
	return nil
}

func copyfiles(trgdir string, fs fs.FS) error {
	if err := os.MkdirAll(trgdir, 0o755); err != nil {
		return err
	}
	return os.CopyFS(trgdir, fs)
}
