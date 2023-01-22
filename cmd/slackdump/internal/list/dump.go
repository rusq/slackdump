package list

import (
	"context"
	_ "embed"
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
)

//go:embed assets/list_conversation.md
var dumpMd string

// CmdListConversation is the command to list conversations.
var CmdListConversation = &base.Command{
	UsageLine:   "slackdump list convo [flags] <conversation list>",
	Short:       "synonym for 'slackdump dump'",
	PrintFlags:  true,
	RequireAuth: true,
	Long:        "",
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
	CmdListConversation.Run = RunDump
	CmdListConversation.Long = HelpDump(CmdListConversation)
	InitDumpFlagset(&CmdListConversation.Flag)
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

	lg := dlog.FromContext(ctx)

	// initialize the file naming template.
	if opts.NameTemplate == "" {
		lg.Print("File name template is empty, using the default.")
		opts.NameTemplate = nametmpl.Default
	}
	nameTemplate, err := nametmpl.Compile(opts.NameTemplate)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("file template error: %w", err)
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

	// Dump conversations.
	for _, link := range list.Include {
		if err := dump(ctx, sess, nameTemplate, extmap[listType], opts, link); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
	}
	return nil
}

// dump dumps the conversation and saves it to the filesystem.  Filesystem is
// specified in the global config.
func dump(ctx context.Context, sess *slackdump.Session, t *template.Template, ext string, opts options, link string) error {
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

	if err := saveData(ctx, sess, conv, buf.String()+"."+ext, listType); err != nil {
		return err
	}
	return nil
}

var helpTmpl = template.Must(template.New("dumphelp").Parse(string(dumpMd)))

// HelpDump returns the help message for the dump command.
func HelpDump(cmd *base.Command) string {
	var buf strings.Builder
	if err := helpTmpl.Execute(&buf, cmd); err != nil {
		panic(err)
	}
	return base.Render(buf.String())
}
