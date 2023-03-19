package export

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rusq/dlog"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/export/expproc"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/structures"
)

var CmdExport = &base.Command{
	Run:         nil,
	Wizard:      nil,
	UsageLine:   "slackdump export",
	Short:       "exports the Slack Workspace or individual conversations",
	Long:        ``, // TODO: add long description
	CustomFlags: false,
	PrintFlags:  true,
	RequireAuth: true,
}

var (
	options = export.Config{
		Type:   export.TStandard,
		Oldest: time.Time(cfg.Oldest),
		Latest: time.Time(cfg.Latest),
	}
)

func init() {
	// TODO: move TimeValue somewhere more appropriate once v1 is sunset.
	CmdExport.Flag.Var(&options.Type, "type", "export type")
	CmdExport.Flag.StringVar(&options.ExportToken, "export-token", "", "file export token to append to each of the file URLs")
}

func init() {
	CmdExport.Run = runExport
	CmdExport.Wizard = wizExport
}

func runExport(ctx context.Context, cmd *base.Command, args []string) error {
	if cfg.SlackConfig.BaseLocation == "" {
		return errors.New("use -base to set the base output location")
	}
	list, err := structures.NewEntityList(args)
	if err != nil {
		return fmt.Errorf("error parsing the entity list: %w", err)
	}

	prov, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}
	sess, err := slackdump.New(ctx, prov, cfg.SlackConfig)
	if err != nil {
		return err
	}
	defer sess.Close()

	options.List = list
	options.Logger = dlog.FromContext(ctx)

	return exportV2(ctx, sess, list, options)
}

func exportV2(ctx context.Context, sess *slackdump.Session, list *structures.EntityList, options export.Config) error {
	exp := export.New(sess, options)
	return exp.Run(ctx)
}

func exportV3(ctx context.Context, sess *slackdump.Session, list *structures.EntityList, options export.Config) error {
	lg := dlog.FromContext(ctx)
	tmpdir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return err
	}

	errC := make(chan error, 1)
	s := sess.Stream()
	var wg sync.WaitGroup
	{
		userproc, err := expproc.NewUsers(tmpdir)
		if err != nil {
			return err
		}
		wg.Add(1)
		go func() {
			errC <- s.Users(ctx, userproc)
			errC <- userproc.Close()
			wg.Done()
			lg.Debug("users done")
		}()
	}
	{
		var channelsC = make(chan []slack.Channel, 1)
		chanproc, err := expproc.NewChannels(tmpdir, func(c []slack.Channel) error {
			channelsC <- c
			return nil
		})
		if err != nil {
			return err
		}

		wg.Add(1)
		go func() {
			errC <- s.ListChannels(ctx, slackdump.AllChanTypes, chanproc)
			errC <- chanproc.Close()
			wg.Done()
			lg.Debug("channels done")
		}()
	}

	panic("not implemented")
}
