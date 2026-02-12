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

package archive

import (
	"context"
	_ "embed"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/directory"
	"github.com/rusq/slackdump/v4/internal/chunk/control"
	"github.com/rusq/slackdump/v4/internal/client"
	"github.com/rusq/slackdump/v4/internal/convert/transform/fileproc"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/source"
	"github.com/rusq/slackdump/v4/stream"
)

//go:embed assets/archive.md
var mdArchive string

var CmdArchive = &base.Command{
	Run:         RunArchive,
	UsageLine:   "slackdump archive [flags] [link1[ link 2[ link N]]]",
	Short:       "archive the workspace or individual conversations on disk",
	Long:        mdArchive,
	FlagMask:    cfg.OmitUserCacheFlag,
	RequireAuth: true,
	PrintFlags:  true,
}

func init() {
	CmdArchive.Wizard = archiveWizard
}

var errNoOutput = errors.New("output directory is required")

func RunArchive(ctx context.Context, cmd *base.Command, args []string) error {
	if cfg.UseChunkFiles {
		return runChunkArchive(ctx, cmd, args)
	} else {
		return runDBArchive(ctx, cmd, args)
	}
}

func runChunkArchive(ctx context.Context, _ *base.Command, args []string) error {
	start := time.Now()
	list, err := structures.NewEntityList(args)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	client, err := bootstrap.Slack(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	if err := bootstrap.AskOverwrite(cfg.Output); err != nil {
		return err
	}

	cd, err := NewDirectory(cfg.Output)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	defer cd.Close()

	ctrl, err := ArchiveController(ctx, cd, client)
	if err != nil {
		return err
	}
	defer ctrl.Close()
	if err := ctrl.RunNoTransform(ctx, list); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	cfg.Log.Info("Recorded workspace data", "directory", cd.Name(), "took", time.Since(start))
	return nil
}

func runDBArchive(ctx context.Context, cmd *base.Command, args []string) error {
	start := time.Now()
	list, err := structures.NewEntityList(args)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	client, err := bootstrap.Slack(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	dirname := cfg.StripZipExt(cfg.Output)
	if err := os.MkdirAll(dirname, 0o755); err != nil {
		return err
	}

	dbfile := filepath.Join(dirname, source.DefaultDBFile)
	if err := bootstrap.AskOverwrite(dbfile); err != nil {
		return err
	}

	conn, err := sqlx.Open(repository.Driver, dbfile)
	if err != nil {
		return err
	}
	defer conn.Close()

	flags := control.Flags{
		MemberOnly:    cfg.MemberOnly,
		RecordFiles:   cfg.RecordFiles,
		ChannelUsers:  cfg.OnlyChannelUsers,
		IncludeLabels: cfg.IncludeCustomLabels,
		ChannelTypes:  cfg.ChannelTypes,
	}

	ctrl, err := DBController(ctx, cmd.Name(), conn, client, dirname, flags, []stream.Option{})
	if err != nil {
		return err
	}
	defer func() {
		if err := ctrl.Close(); err != nil {
			slog.ErrorContext(ctx, "unable to close database controller", "error", err)
		}
	}()

	if err := ctrl.RunNoTransform(ctx, list); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	cfg.Log.Info("Recorded workspace data", "directory", dirname, "took", time.Since(start))

	return nil
}

// NewDirectory creates a new chunk directory with name.  If name has a .zip
// extension it is stripped.
func NewDirectory(name string) (*chunk.Directory, error) {
	name = cfg.StripZipExt(name)
	if name == "" {
		return nil, errNoOutput
	}

	cd, err := chunk.CreateDir(name)
	if err != nil {
		return nil, err
	}
	return cd, nil
}

// DBController returns a new database controller initialised with the given
// parameters.
//
// Obscene, just obscene amount of arguments.
func DBController(ctx context.Context, cmdName string, conn *sqlx.DB, client client.Slack, dirname string, flags control.Flags, streamOpts []stream.Option, opts ...dbase.Option) (Controller, error) {
	lg := cfg.Log
	dbp, err := dbase.New(ctx, conn, bootstrap.SessionInfo(cmdName), opts...)
	if err != nil {
		return nil, err
	}
	sopts := []stream.Option{
		stream.OptLatest(time.Time(cfg.Latest)),
		stream.OptOldest(time.Time(cfg.Oldest)),
		stream.OptResultFn(resultLogger(lg)),
	}
	sopts = append(sopts, streamOpts...)
	// start attachment downloader
	dl := fileproc.NewDownloader(
		ctx,
		cfg.WithFiles,
		client,
		fsadapter.NewDirectory(dirname),
		lg,
	)
	// start avatar downloader
	avdl := fileproc.NewDownloader(
		ctx,
		cfg.WithAvatars,
		client,
		fsadapter.NewDirectory(dirname),
		lg,
	)

	ctrl, err := control.New(
		ctx,
		stream.New(client, cfg.Limits, sopts...),
		dbp,
		control.WithFiler(fileproc.New(dl)),
		control.WithAvatarProcessor(fileproc.NewAvatarProc(avdl)),
		control.WithFlags(flags),
	)
	if err != nil {
		return nil, err
	}
	return ctrl, nil
}

type Controller interface {
	// Run should run the main controller loop.
	Run(context.Context, *structures.EntityList) error
	// RunNoTransform should run the main controller loop without
	// enabling transformation logic.
	RunNoTransform(context.Context, *structures.EntityList) error

	io.Closer
}

// ArchiveController returns the default archive controller initialised based
// on global configuration parameters.
func ArchiveController(ctx context.Context, cd *chunk.Directory, client client.Slack, opts ...stream.Option) (*control.Controller, error) {
	lg := cfg.Log

	sopts := []stream.Option{
		stream.OptLatest(time.Time(cfg.Latest)),
		stream.OptOldest(time.Time(cfg.Oldest)),
		stream.OptResultFn(resultLogger(lg)),
	}
	sopts = append(sopts, opts...)

	// start attachment downloader
	dl := fileproc.NewDownloader(
		ctx,
		cfg.WithFiles,
		client,
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)
	// start avatar downloader
	avdl := fileproc.NewDownloader(
		ctx,
		cfg.WithAvatars,
		client,
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)

	erc := directory.NewERC(cd, lg)

	flags := control.Flags{
		MemberOnly:    cfg.MemberOnly,
		RecordFiles:   cfg.RecordFiles,
		ChannelUsers:  cfg.OnlyChannelUsers,
		IncludeLabels: cfg.IncludeCustomLabels,
		ChannelTypes:  cfg.ChannelTypes,
	}

	ctrl, err := control.New(
		ctx,
		stream.New(client, cfg.Limits, sopts...),
		erc,
		control.WithLogger(lg),
		control.WithFlags(flags),
		control.WithFiler(fileproc.New(dl)),
		control.WithAvatarProcessor(fileproc.NewAvatarProc(avdl)),
	)
	if err != nil {
		return nil, err
	}

	return ctrl, nil
}

func resultLogger(lg *slog.Logger) func(sr stream.Result) error {
	return func(sr stream.Result) error {
		lg.Info("stream", "result", sr.String())
		return nil
	}
}
