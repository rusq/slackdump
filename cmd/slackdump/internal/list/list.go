package list

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/convert/format"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
)

var CmdList = &base.Command{
	UsageLine: "slackdump list",
	Short:     "list users or channels",
	Long: `
List lists users or channels for the Slack Workspace.  It may take a while on
large workspaces, as Slack limits the amount of requests on it's own discretion,
which is sometimes unreasonably slow.
`,
	Commands: []*base.Command{
		CmdListUsers,
		CmdListChannels,
	},
}

// common flags
var listType format.Type

func addCommonFlags(fs *flag.FlagSet) {
	fs.Var(&listType, "type", fmt.Sprintf("list type, if not specified will use JSON.  Supported values: %v", format.AllTypes))
}

func init() {
	for _, cmd := range CmdList.Commands {
		addCommonFlags(&cmd.Flag)
	}
}

func serialise(fs fsadapter.FS, name string, a any) error {
	f, err := fs.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if err := enc.Encode(a); err != nil {
		return err
	}
	return nil
}

type listFunc func(ctx context.Context, sess *slackdump.Session) (a any, filename string, err error)

func list(ctx context.Context, listFn listFunc) error {
	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}
	sess, err := slackdump.NewWithOptions(ctx, prov, cfg.SlackOptions)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	a, filename, err := listFn(ctx, sess)
	if err != nil {
		return err
	}
	if listType != format.CUnknown {
		// stdout output
		if err := fmtPrint(ctx, os.Stdout, a, listType, sess.Users); err != nil {
			return err
		}
	} else {
		// save JSON
		fs, err := fsadapter.New(cfg.BaseLoc)
		if err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
		defer fs.Close()
		if err := serialise(fs, filename, a); err != nil {
			return err
		}
	}

	return nil
}

func fmtPrint(ctx context.Context, w io.Writer, a any, typ format.Type, u []slack.User) error {
	initFn, ok := format.Converters[typ]
	if !ok {
		return fmt.Errorf("unknown converter type: %s", typ)
	}
	cvt := initFn()
	switch val := a.(type) {
	case types.Channels:
		return cvt.Channels(ctx, w, u, val)
	case types.Users:
		return cvt.Users(ctx, w, val)
	default:
		return fmt.Errorf("unknown data type: %T", a)
	}
	// unreachable
}
