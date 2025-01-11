package dump

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
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
	"github.com/rusq/slackdump/v3/downloader"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/nametmpl"
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
		downloadFiles: cfg.DownloadFiles,
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
		return errors.New("no filesystem adapter")
	}
	if p.list.IsEmpty() {
		return ErrNothingToDo
	}
	if err := p.validate(); err != nil {
		return err
	}

	dir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	lg := cfg.Log
	lg.Debug("using directory", "dir", dir)

	// files subprocessor
	var sdl fileproc.Downloader
	if p.downloadFiles {
		dl := downloader.New(sess.Client(), fsa, downloader.WithLogger(lg))
		if err := dl.Start(ctx); err != nil {
			return err
		}
		defer dl.Stop()
		sdl = dl
	} else {
		sdl = fileproc.NoopDownloader{}
	}

	subproc := fileproc.NewDumpSubproc(sdl)

	opts := []transform.StdOption{
		transform.StdWithTemplate(p.tmpl),
		transform.StdWithLogger(lg),
	}
	if p.updatePath && p.downloadFiles {
		opts = append(opts, transform.StdWithPipeline(subproc.PathUpdateFunc))
	}

	// Initialise the standard transformer.
	cd, err := chunk.OpenDir(dir)
	if err != nil {
		return err
	}
	defer cd.Close()

	tf, err := transform.NewStandard(fsa, cd, opts...)
	if err != nil {
		return fmt.Errorf("failed to create transform: %w", err)
	}

	coord := transform.NewCoordinator(ctx, tf)

	// Create conversation processor.
	proc, err := dirproc.NewConversation(cd, subproc, coord, dirproc.WithLogger(lg), dirproc.WithRecordFiles(false))
	if err != nil {
		return fmt.Errorf("failed to create conversation processor: %w", err)
	}
	defer func() {
		if err := proc.Close(); err != nil {
			lg.WarnContext(ctx, "failed to close conversation processor", "error", err)
		}
	}()

	if err := sess.Stream(
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
	).Conversations(ctx, proc, p.list.C(ctx)); err != nil {
		return fmt.Errorf("failed to dump conversations: %w", err)
	}

	lg.DebugContext(ctx, "stream complete, waiting for all goroutines to finish")
	if err := coord.Wait(); err != nil {
		return err
	}

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
