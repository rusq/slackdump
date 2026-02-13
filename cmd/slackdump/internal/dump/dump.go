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

package dump

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime/trace"
	"strings"
	"text/template"
	"time"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v4/internal/chunk/control"
	"github.com/rusq/slackdump/v4/internal/client"
	"github.com/rusq/slackdump/v4/internal/convert/transform"
	"github.com/rusq/slackdump/v4/internal/convert/transform/fileproc"
	"github.com/rusq/slackdump/v4/internal/nametmpl"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/source"
	"github.com/rusq/slackdump/v4/stream"
)

//go:embed assets/list_conversation.md
var dumpMd string

// CmdDump is the dump command.
var CmdDump = &base.Command{
	UsageLine:   "slackdump dump [flags] <IDs or URLs>",
	Short:       "dump individual conversations or threads",
	Long:        dumpMd,
	RequireAuth: true,
	PrintFlags:  true,
	FlagMask: (cfg.OmitCustomUserFlags |
		cfg.OmitRecordFilesFlag |
		cfg.OmitWithAvatarsFlag |
		cfg.OmitChannelTypesFlag), // we don't need channel types, as dump requires explicit channel ids
}

func init() {
	CmdDump.Run = RunDump
	CmdDump.Wizard = WizDump
	CmdDump.Long = helpDump(CmdDump)
}

// ErrNothingToDo is returned if there are no links to dump.
var ErrNothingToDo = errors.New("no conversations to dump, run \"slackdump help dump\"")

type options struct {
	nameTemplate string // NameTemplate is the template for the output file name.
	updateLinks  bool   // update file links to point to the downloaded files
}

var opts options

// initDumpFlagset initializes the flagset for the dump command.
func initDumpFlagset(fs *flag.FlagSet) {
	fs.StringVar(&opts.nameTemplate, "ft", nametmpl.Default, "output file naming template.\n")
	fs.BoolVar(&opts.updateLinks, "update-links", false, "update file links to point to the downloaded files.")
}

func init() {
	initDumpFlagset(&CmdDump.Flag)
}

// RunDump is the main entry point for the dump command.
func RunDump(ctx context.Context, _ *base.Command, args []string) error {
	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return ErrNothingToDo
	}

	lg := cfg.Log

	// initialize the list of entities to dump.
	list, err := structures.NewEntityList(args)
	if err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	} else if list.IsEmpty() {
		base.SetExitStatus(base.SInvalidParameters)
		return ErrNothingToDo
	}

	// initialize the file naming template.
	if opts.nameTemplate == "" {
		opts.nameTemplate = nametmpl.Default
	}
	tmpl, err := nametmpl.New(opts.nameTemplate)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("file template error: %w", err)
	}
	if err := bootstrap.AskOverwrite(cfg.Output); err != nil {
		return err
	}

	fsa, err := fsadapter.New(cfg.Output)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer func() {
		if err := fsa.Close(); err != nil {
			lg.WarnContext(ctx, "warning: failed to close the filesystem", "error", err)
		}
	}()

	client, err := bootstrap.Slack(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	p := dumpparams{
		list:          list,
		tmpl:          tmpl,
		updatePath:    opts.updateLinks,
		downloadFiles: cfg.WithFiles,
	}

	// leave the compatibility mode to the user, if the new version is playing
	// tricks.
	start := time.Now()
	if err := dump(ctx, client, fsa, p); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	lg.InfoContext(ctx, "conversation dump finished", "output", cfg.Output, "count", p.list.IncludeCount(), "took", time.Since(start))
	return nil
}

type dumpparams struct {
	list          *structures.EntityList // list of entities to dump
	tmpl          *nametmpl.Template     // file naming template
	updatePath    bool                   // update filepath to point to the downloaded file?
	downloadFiles bool                   // download files?
}

func (p *dumpparams) validate() error {
	if p.list.IsEmpty() {
		return ErrNothingToDo
	}
	if p.tmpl == nil {
		p.tmpl = nametmpl.NewDefault()
	}
	return nil
}

// dump generates the files in slackdump format.
func dump(ctx context.Context, client client.Slack, fsa fsadapter.FS, p dumpparams) error {
	// it uses Stream to generate a chunk file, then process it and generate
	// dump JSON.
	ctx, task := trace.NewTask(ctx, "dump")
	defer task.End()

	if fsa == nil {
		return errors.New("internal error:  no filesystem adapter")
	}
	if p.list.IsEmpty() {
		return ErrNothingToDo
	}
	if err := p.validate(); err != nil {
		return err
	}

	backend, err := newDumpBackend(ctx)
	if err != nil {
		return err
	}
	defer backend.cleanup()

	return runDump(ctx, backend, client, fsa, p)
}

var helpTmpl = template.Must(template.New("dumphelp").Parse(dumpMd))

// helpDump returns the help message for the dump command.
func helpDump(cmd *base.Command) string {
	var buf strings.Builder
	if err := helpTmpl.Execute(&buf, cmd); err != nil {
		panic(err)
	}
	return buf.String()
}

type dumpController interface {
	Run(context.Context, *structures.EntityList) error
	Close() error
}

type dumpBackend struct {
	src     source.Sourcer
	build   func(context.Context, *stream.Stream, fileproc.FileProcessor, *transform.Coordinator) (dumpController, error)
	cleanup func()
}

func newDumpBackend(ctx context.Context) (dumpBackend, error) {
	if cfg.UseChunkFiles {
		return newChunkBackend(ctx)
	}
	return newDatabaseBackend(ctx)
}

// runDump contains the common logic that was duplicated in dumpv3/dumpv31.
func runDump(ctx context.Context, backend dumpBackend, client client.Slack, fsa fsadapter.FS, p dumpparams) error {
	lg := cfg.Log

	sdl := fileproc.NewDownloader(ctx, p.downloadFiles, client, fsa, lg)
	subproc := fileproc.NewDump(sdl)
	defer subproc.Close()

	opts := []transform.DumpOption{
		transform.DumpWithTemplate(p.tmpl),
		transform.DumpWithLogger(lg),
	}
	if p.updatePath && p.downloadFiles {
		opts = append(opts, transform.DumpWithPipeline(subproc.PathUpdateFunc))
	}

	tf, err := transform.NewDump(fsa, backend.src, opts...)
	if err != nil {
		return fmt.Errorf("failed to create transform: %w", err)
	}
	coord := transform.NewCoordinator(ctx, tf)

	str := stream.New(client, cfg.Limits,
		stream.OptOldest(time.Time(cfg.Oldest)),
		stream.OptLatest(time.Time(cfg.Latest)),
		stream.OptResultFn(func(sr stream.Result) error {
			if sr.Err != nil {
				return sr.Err
			}
			if sr.IsLast {
				lg.InfoContext(ctx, "dumped", "sr", sr.String())
			}
			return nil
		}),
	)

	ctrl, err := backend.build(ctx, str, subproc, coord)
	if err != nil {
		return fmt.Errorf("error creating controller: %w", err)
	}
	defer ctrl.Close()

	if err := ctrl.Run(ctx, p.list); err != nil {
		return fmt.Errorf("failed to run controller: %w", err)
	}
	if err := coord.Wait(); err != nil {
		return fmt.Errorf("failed to wait for coordinator: %w", err)
	}
	return nil
}

func newChunkBackend(ctx context.Context) (dumpBackend, error) {
	lg := cfg.Log
	dir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return dumpBackend{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	lg.Debug("using directory", "dir", dir)

	cd, err := chunk.OpenDir(dir)
	if err != nil {
		return dumpBackend{}, err
	}

	return dumpBackend{
		src: source.OpenChunkDir(cd, true),
		build: func(_ context.Context, s *stream.Stream, subproc fileproc.FileProcessor, coord *transform.Coordinator) (dumpController, error) {
			ctrl := control.NewDir(
				cd,
				s,
				control.WithLogger(lg),
				control.WithFlags(control.Flags{RecordFiles: cfg.WithFiles}),
				control.WithCoordinator(coord),
				control.WithFiler(subproc),
			)
			return ctrl, nil
		},
		cleanup: func() {
			if err := cd.Close(); err != nil {
				lg.ErrorContext(ctx, "unable to close chunk directory", "error", err)
			}
		},
	}, nil
}

func newDatabaseBackend(ctx context.Context) (dumpBackend, error) {
	lg := cfg.Log

	tmpdir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return dumpBackend{}, fmt.Errorf("failed to create temp directory: %w", err)
	}

	wconn, si, err := bootstrap.Database(tmpdir, "dump")
	if err != nil {
		_ = os.RemoveAll(tmpdir)
		return dumpBackend{}, err
	}
	tmpdbp, err := dbase.New(ctx, wconn, si)
	if err != nil {
		_ = wconn.Close()
		_ = os.RemoveAll(tmpdir)
		return dumpBackend{}, err
	}
	lg.DebugContext(ctx, "using database in ", "dir", tmpdir)

	removeTmp := !lg.Enabled(ctx, slog.LevelDebug)

	return dumpBackend{
		src: source.DatabaseWithSource(tmpdbp.Source()),
		build: func(ctx context.Context, s *stream.Stream, subproc fileproc.FileProcessor, coord *transform.Coordinator) (dumpController, error) {
			return control.New(
				ctx,
				s,
				tmpdbp,
				control.WithLogger(lg),
				control.WithFlags(control.Flags{RecordFiles: cfg.WithFiles}),
				control.WithCoordinator(coord),
				control.WithFiler(subproc),
			)
		},
		cleanup: func() {
			if err := tmpdbp.Close(); err != nil {
				lg.ErrorContext(ctx, "unable to close database processor", "error", err)
			}
			if err := wconn.Close(); err != nil {
				lg.ErrorContext(ctx, "unable to close database connection", "error", err)
			}
			if removeTmp {
				if err := os.RemoveAll(tmpdir); err != nil {
					lg.ErrorContext(ctx, "unable to remove temporary directory", "dir", tmpdir, "error", err)
				}
			}
		},
	}, nil
}
