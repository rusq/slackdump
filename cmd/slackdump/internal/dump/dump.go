package dump

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"time"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/app/config"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

//go:embed assets/dump.md
var mdDump string

var CmdDump = &base.Command{
	UsageLine:   "slackdump dump [flags] <IDs or URLs>",
	Short:       "dump individual conversations or threads",
	Long:        base.Render(mdDump),
	RequireAuth: true,
	PrintFlags:  true,
}

var ErrNothingToDo = errors.New("no conversations to dump, seek help")

type options struct {
	Oldest       time.Time
	Latest       time.Time
	NameTemplate string
}

var opts options

func ptr[T any](a T) *T {
	return &a
}

func init() {
	CmdDump.Run = runDump
	CmdDump.Flag.Var(ptr(config.TimeValue(opts.Oldest)), "from", "timestamp of the oldest message to fetch")
	CmdDump.Flag.Var(ptr(config.TimeValue(opts.Latest)), "to", "timestamp of the newest message to fetch")
	CmdDump.Flag.StringVar(&opts.NameTemplate, "ft", "{{.ID}}{{ if .ThreadTS}}-{{.ThreadTS}}{{end}}", "output file naming template.")
}

func runDump(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return ErrNothingToDo
	}

	list, err := structures.NewEntityList(args)
	if err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}
	if list.IsEmpty() {
		base.SetExitStatus(base.SInvalidParameters)
		return ErrNothingToDo
	}

	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	if fs, err := fsadapter.New(cfg.BaseLoc); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	} else {
		cfg.SlackOptions.Filesystem = fs
		defer fsadapter.Close(fs)
	}

	sess, err := slackdump.NewWithOptions(ctx, prov, cfg.SlackOptions)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	for _, link := range list.Include {
		conv, err := sess.Dump(ctx, link, opts.Oldest, opts.Latest)
		if err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
		if err := saveConversation(cfg.SlackOptions.Filesystem, "test.json", conv); err != nil {
			return err
		}

	}
	return nil
}

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
