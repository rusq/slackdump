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

//go:embed assets/list_conversation.md
var dumpMd string

const codeBlock = "```"

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
	Oldest       time.Time // Oldest is the timestamp of the oldest message to fetch.
	Latest       time.Time // Latest is the timestamp of the newest message to fetch.
	NameTemplate string    // NameTemplate is the template for the output file name.
	JSONL        bool      // JSONL should be true if the output should be JSONL instead of JSON.
}

var opts options

// ptr returns a pointer to the given value.
func ptr[T any](a T) *T { return &a }

// InitDumpFlagset initializes the flagset for the dump command.
func InitDumpFlagset(fs *flag.FlagSet) {
	fs.Var(ptr(config.TimeValue(opts.Oldest)), "from", "timestamp of the oldest message to fetch")
	fs.Var(ptr(config.TimeValue(opts.Latest)), "to", "timestamp of the newest message to fetch")
	fs.StringVar(&opts.NameTemplate, "ft", nametmpl.Default, "output file naming template.\n")
	fs.BoolVar(&opts.JSONL, "jsonl", false, "output JSONL instead of JSON")
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
	if opts.NameTemplate == "" {
		opts.NameTemplate = nametmpl.Default
	}
	namer, err := newNamer(opts.NameTemplate, "json")
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return fmt.Errorf("file template error: %w", err)
	}

	// Initialize the filesystem.
	fs, err := fsadapter.New(cfg.BaseLoc)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer fs.Close()

	// Initialize the session.
	cfg.SlackOptions.Filesystem = fs
	cfg.SlackOptions.Logger = dlog.FromContext(ctx)

	sess, err := slackdump.New(ctx, prov, cfg.SlackOptions)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	// Dump conversations.
	for _, link := range list.Include {
		conv, err := sess.Dump(ctx, link, opts.Oldest, opts.Latest)
		if err != nil {
			return err
		}

		if err := save(ctx, fs, namer.Filename(conv), conv); err != nil {
			return err
		}
	}
	return nil
}

// namer is a helper type to generate filenames for conversations.
type namer struct {
	t   *template.Template
	ext string
}

// newNamer returns a new namer.  It must be called with a valid template.
func newNamer(tmpl string, ext string) (namer, error) {
	t, err := template.New("name").Parse(tmpl)
	if err != nil {
		return namer{}, err
	}
	return namer{t: t, ext: ext}, nil
}

// Filename returns the filename for the given conversation.
func (n namer) Filename(conv *types.Conversation) string {
	var buf strings.Builder
	if err := n.t.Execute(&buf, conv); err != nil {
		panic(err)
	}
	return buf.String() + "." + n.ext
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
