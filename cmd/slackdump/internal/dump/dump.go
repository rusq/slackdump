package dump

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime/trace"
	"strings"
	"text/template"
	"time"

	"github.com/rusq/dlog"
	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v2/internal/chunk/transform"
	"github.com/rusq/slackdump/v2/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v2/internal/nametmpl"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/rusq/slackdump/v2/types"
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
}

func init() {
	CmdDump.Run = RunDump
	CmdDump.Wizard = WizDump
	CmdDump.Long = HelpDump(CmdDump)
}

// ErrNothingToDo is returned if there's no links to dump.
var ErrNothingToDo = errors.New("no conversations to dump, run \"slackdump help dump\"")

type options struct {
	nameTemplate string // NameTemplate is the template for the output file name.
	compat       bool   // compatibility mode
	updateLinks  bool   // update file links to point to the downloaded files
}

var opts options

// InitDumpFlagset initializes the flagset for the dump command.
func InitDumpFlagset(fs *flag.FlagSet) {
	fs.StringVar(&opts.nameTemplate, "ft", nametmpl.Default, "output file naming template.\n")
	fs.BoolVar(&opts.compat, "compat", false, "compatibility mode")
	fs.BoolVar(&opts.updateLinks, "update-links", false, "update file links to point to the downloaded files.")
}

func init() {
	InitDumpFlagset(&CmdDump.Flag)
}

// RunDump is the main entry point for the dump command.
func RunDump(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return ErrNothingToDo
	}
	// Retrieve the Authentication provider.
	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

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

	fsa, err := fsadapter.New(cfg.BaseLocation)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer fsa.Close()

	sess, err := slackdump.New(ctx, prov, slackdump.WithLogger(logger.FromContext(ctx)), slackdump.WithFilesystem(fsa))
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	p := dumpparams{
		list:      list,
		tmpl:      tmpl,
		fpupdate:  opts.updateLinks,
		dumpFiles: cfg.DumpFiles,
		oldest:    time.Time(cfg.Oldest),
		latest:    time.Time(cfg.Latest),
	}

	// leave the compatibility mode to the user, if the new version is playing
	// tricks.
	start := time.Now()
	dumpFn := dump
	if opts.compat {
		dumpFn = dumpv2
	}
	if err := dumpFn(ctx, sess, fsa, p); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	dlog.FromContext(ctx).Printf("dumped %d conversations in %s", len(p.list.Include), time.Since(start))
	return nil
}

type dumpparams struct {
	list      *structures.EntityList // list of entities to dump
	tmpl      *nametmpl.Template     // file naming template
	oldest    time.Time
	latest    time.Time
	fpupdate  bool // update filepath?
	dumpFiles bool // download files?
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

func dump(ctx context.Context, sess *slackdump.Session, fsa fsadapter.FS, p dumpparams) error {
	ctx, task := trace.NewTask(ctx, "dump")
	defer task.End()

	if err := p.validate(); err != nil {
		return err
	}
	if fsa == nil {
		return fmt.Errorf("no filesystem adapter")
	}

	lg := logger.FromContext(ctx)

	if p.list.IsEmpty() {
		return ErrNothingToDo
	}

	dir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	lg.Debugf("using directory: %s", dir)

	// files subprocessor
	var sdl fileproc.Downloader
	if p.dumpFiles {
		dl := downloader.New(sess.Client(), fsa, downloader.WithLogger(lg))
		dl.Start(ctx)
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
	if p.fpupdate && p.dumpFiles {
		opts = append(opts, transform.StdWithPipeline(subproc.PathUpdateFunc))
	}

	// Initialise the standard transformer.
	tf, err := transform.NewStandard(fsa, dir, opts...)
	if err != nil {
		return fmt.Errorf("failed to create transform: %w", err)
	}

	coord := transform.NewCoordinator(ctx, tf)

	cd, err := chunk.OpenDir(dir)
	if err != nil {
		return err
	}
	// Create conversation processor.
	proc, err := dirproc.NewConversation(cd, subproc, coord, dirproc.WithLogger(lg), dirproc.WithRecordFiles(false))
	if err != nil {
		return fmt.Errorf("failed to create conversation processor: %w", err)
	}
	defer func() {
		if err := proc.Close(); err != nil {
			lg.Printf("failed to close conversation processor: %v", err)
		}
	}()

	if err := sess.Stream(
		slackdump.OptOldest(time.Time(p.oldest)),
		slackdump.OptLatest(time.Time(p.latest)),
		slackdump.OptResultFn(func(sr slackdump.StreamResult) error {
			if sr.Err != nil {
				return sr.Err
			}
			dlog.Printf("conversation %s dumped", sr)
			return nil
		}),
	).Conversations(ctx, proc, p.list.Generator(ctx)); err != nil {
		return fmt.Errorf("failed to dump conversations: %w", err)
	}

	lg.Debugln("stream complete, waiting for all goroutines to finish")
	if err := coord.Wait(); err != nil {
		return err
	}

	return nil
}

func dumpv2(ctx context.Context, sess *slackdump.Session, fs fsadapter.FS, p dumpparams) error {
	for _, link := range p.list.Include {
		conv, err := sess.Dump(ctx, link, time.Time(p.oldest), time.Time(p.latest))
		if err != nil {
			return err
		}
		if err := save(ctx, fs, p.tmpl.Execute(conv), conv); err != nil {
			return err
		}
	}
	return nil
}

func save(ctx context.Context, fs fsadapter.FS, filename string, conv *types.Conversation) error {
	_, task := trace.NewTask(ctx, "saveData")
	defer task.End()

	f, err := fs.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(conv)
}

var helpTmpl = template.Must(template.New("dumphelp").Parse(string(dumpMd)))

// HelpDump returns the help message for the dump command.
func HelpDump(cmd *base.Command) string {
	var buf strings.Builder
	if err := helpTmpl.Execute(&buf, cmd); err != nil {
		panic(err)
	}
	return buf.String()
}
