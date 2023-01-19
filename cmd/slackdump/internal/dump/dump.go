package dump

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"runtime/trace"
	"strings"
	"text/template"
	"time"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/app/config"
	"github.com/rusq/slackdump/v2/internal/app/nametmpl"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

//go:embed assets/dump.md
var dumpMd string

// CmdDump is the dump command.
var CmdDump = &base.Command{
	UsageLine:   "slackdump dump [flags] <IDs or URLs>",
	Short:       "dump individual conversations or threads",
	Long:        base.Render(dumpMd),
	RequireAuth: true,
	PrintFlags:  true,
}

// ErrNothingToDo is returned if there's no links to dump.
var ErrNothingToDo = errors.New("no conversations to dump, run \"slackdump help dump\"")

type options struct {
	Oldest       time.Time // Oldest is the timestamp of the oldest message to fetch.
	Latest       time.Time // Latest is the timestamp of the newest message to fetch.
	NameTemplate string    // NameTemplate is the template for the output file name.
}

var opts options

// ptr returns a pointer to the given value.
func ptr[T any](a T) *T { return &a }

func init() {
	CmdDump.Run = RunDump
	InitDumpFlagset(&CmdDump.Flag)
}

func InitDumpFlagset(fs *flag.FlagSet) {
	fs.Var(ptr(config.TimeValue(opts.Oldest)), "from", "timestamp of the oldest message to fetch")
	fs.Var(ptr(config.TimeValue(opts.Latest)), "to", "timestamp of the newest message to fetch")
	fs.StringVar(&opts.NameTemplate, "ft", nametmpl.Default, "output file naming template.\n")
}

// RunDump is the main entry point for the dump command.
func RunDump(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return ErrNothingToDo
	}

	lg := dlog.FromContext(ctx)

	// initialize the file name template.
	if opts.NameTemplate == "" {
		lg.Print("File name template is empty, using the default.")
		opts.NameTemplate = nametmpl.Default
	}
	nameTemplate, err := nametmpl.Compile(opts.NameTemplate)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("file template error: %w", err)
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

	// Retrieve the Authentication provider.
	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	// Initialize the filesystem.
	if fs, err := fsadapter.New(cfg.BaseLoc); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	} else {
		cfg.SlackOptions.Filesystem = fs
		defer fs.Close()
	}

	// Initialize the session.
	sess, err := slackdump.NewWithOptions(ctx, prov, cfg.SlackOptions)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	// Dump the conversations.
	for _, link := range list.Include {
		if err := dump(ctx, sess, nameTemplate, opts, link); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
	}
	return nil
}

// dump dumps the conversation and saves it to the filesystem.  Filesystem is
// specified in the global config.
func dump(ctx context.Context, sess *slackdump.Session, t *template.Template, opts options, link string) error {
	ctx, task := trace.NewTask(ctx, "dump")
	defer task.End()

	conv, err := sess.Dump(ctx, link, opts.Oldest, opts.Latest)
	if err != nil {
		return err
	}
	var buf strings.Builder
	if err := t.Execute(&buf, conv); err != nil {
		return err
	}
	if err := saveConversation(cfg.SlackOptions.Filesystem, buf.String()+".json", conv); err != nil {
		return err
	}
	return nil
}

// saveConversation saves the conversation to the filesystem.
func saveConversation(fs fsadapter.FS, filename string, conv *types.Conversation) error {
	f, err := fs.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(conv); err != nil {
		return err
	}
	return nil
}
