package list

import (
	"context"
	"encoding/json"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/fsadapter"
)

var CmdList = &base.Command{
	Run:       runList,
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

func runList(ctx context.Context, cmd *base.Command, args []string) error {
	cmd.Flag.Usage()
	return nil
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
	fs, err := fsadapter.New(cfg.BaseLoc)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer fs.Close()

	a, filename, err := listFn(ctx, sess)
	if err != nil {
		return err
	}
	if err := serialise(fs, filename, a); err != nil {
		return err
	}

	return nil
}
