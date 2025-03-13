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

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/nametmpl"
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/stream"
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
	FlagMask:    cfg.OmitMemberOnlyFlag | cfg.OmitRecordFilesFlag | cfg.OmitDownloadAvatarsFlag,
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
	tmpl, err := nametmpl.New(opts.nameTemplate + ".json")
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("file template error: %w", err)
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

	sess, err := bootstrap.SlackdumpSession(ctx, slackdump.WithFilesystem(fsa))
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
	if err := dump(ctx, sess, fsa, p); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	lg.InfoContext(ctx, "conversation dump finished", "count", p.list.IncludeCount(), "took", time.Since(start))
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
func dump(ctx context.Context, sess *slackdump.Session, fsa fsadapter.FS, p dumpparams) error {
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

	if cfg.UseChunkFiles {
		return dumpv3(ctx, sess, fsa, p)
	} else {
		return dumpv31(ctx, sess, fsa, p)
	}
}

func dumpv3(ctx context.Context, sess *slackdump.Session, fsa fsadapter.FS, p dumpparams) error {
	dir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	lg := cfg.Log
	lg.Debug("using directory", "dir", dir)

	// Initialise the standard transformer.
	cd, err := chunk.OpenDir(dir)
	if err != nil {
		return err
	}
	defer cd.Close()
	src := source.OpenChunkDir(cd, true)

	// files subprocessor
	sdl := fileproc.NewDownloader(ctx, p.downloadFiles, sess.Client(), fsa, cfg.Log)
	subproc := fileproc.NewDump(sdl)
	defer subproc.Close()

	opts := []transform.DumpOption{
		transform.DumpWithTemplate(p.tmpl),
		transform.DumpWithLogger(lg),
	}
	if p.updatePath && p.downloadFiles {
		opts = append(opts, transform.DumpWithPipeline(subproc.PathUpdateFunc))
	}

	tf, err := transform.NewDumpConverter(fsa, src, opts...)
	if err != nil {
		return fmt.Errorf("failed to create transform: %w", err)
	}

	coord := transform.NewCoordinator(ctx, tf)

	// TODO: use export controller
	s := sess.Stream(
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
	ctrl := control.NewDir(cd,
		s,
		control.WithLogger(lg),
		control.WithFlags(control.Flags{RecordFiles: cfg.WithFiles}),
		control.WithCoordinator(coord),
		control.WithFiler(subproc),
	)
	if err := ctrl.Run(ctx, p.list); err != nil {
		return fmt.Errorf("failed to run controller: %w", err)
	}
	if err := coord.Wait(); err != nil {
		return fmt.Errorf("failed to wait for coordinator: %w", err)
	}

	lg.DebugContext(ctx, "stream complete, waiting for all goroutines to finish")

	return nil
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

func dumpv31(ctx context.Context, sess *slackdump.Session, fsa fsadapter.FS, p dumpparams) error {
	lg := cfg.Log

	tmpdir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	if !lg.Enabled(ctx, slog.LevelDebug) {
		defer func() {
			if err := os.RemoveAll(tmpdir); err != nil {
				lg.ErrorContext(ctx, "unable to remove temporary directory", "dir", tmpdir, "error", err)
			}
		}()
	}

	// creating a temporary database to hold the data for the converter.
	wconn, si, err := bootstrap.Database(tmpdir, "dump")
	if err != nil {
		return err
	}
	defer wconn.Close()
	tmpdbp, err := dbproc.New(ctx, wconn, si)
	if err != nil {
		return err
	}
	defer func() {
		if err := tmpdbp.Close(); err != nil {
			lg.ErrorContext(ctx, "unable to close database processor", "error", err)
		}
	}()
	lg.DebugContext(ctx, "using database in ", "dir", tmpdir)
	src := source.DatabaseWithSource(tmpdbp.Source())

	// files subprocessor
	sdl := fileproc.NewDownloader(ctx, p.downloadFiles, sess.Client(), fsa, cfg.Log)
	subproc := fileproc.NewDump(sdl)
	defer subproc.Close()

	opts := []transform.DumpOption{
		transform.DumpWithTemplate(p.tmpl),
		transform.DumpWithLogger(lg),
	}
	if p.updatePath && p.downloadFiles {
		opts = append(opts, transform.DumpWithPipeline(subproc.PathUpdateFunc))
	}

	tf, err := transform.NewDumpConverter(fsa, src, opts...)
	if err != nil {
		return fmt.Errorf("failed to create transform: %w", err)
	}
	coord := transform.NewCoordinator(ctx, tf)

	stream := sess.Stream(
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

	ctrl, err := control.New(
		ctx,
		stream,
		tmpdbp,
		control.WithLogger(lg),
		control.WithFlags(control.Flags{RecordFiles: cfg.WithFiles}),
		control.WithCoordinator(coord),
		control.WithFiler(subproc),
	)
	if err != nil {
		return fmt.Errorf("error creating db controller: %w", err)
	}
	defer ctrl.Close()

	if err := ctrl.Run(ctx, p.list); err != nil {
		return fmt.Errorf("error running db controller: %w", err)
	}
	if err := coord.Wait(); err != nil {
		return fmt.Errorf("error waiting for coordinator: %w", err)
	}
	return nil
}
