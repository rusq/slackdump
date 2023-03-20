package export

import (
	"context"
	"errors"
	"fmt"
	"log"
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

	return exportV3(ctx, sess, list, options)
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
	log.Printf("using %s as the temporary directory", tmpdir)

	errC := make(chan error, 1)
	s := sess.Stream()
	var wg sync.WaitGroup

	// Generator of channel IDs.
	links := make(chan string)
	{
		chanproc, err := expproc.NewChannels(tmpdir, func(c []slack.Channel) error {
			for _, ch := range c {
				// TODO: if ch.IsMember { // only channels that the user is a member of
				select {
				case <-ctx.Done():
					return ctx.Err()
				case links <- ch.ID:
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(links)

			if err := s.ListChannels(ctx, slackdump.AllChanTypes, chanproc); err != nil {
				errC <- fmt.Errorf("error listing channels: %w", err)
				return
			}
			if err := chanproc.Close(); err != nil {
				errC <- fmt.Errorf("error closing channel processor: %w", err)
				return
			}
			lg.Debug("channels done")
		}()
	}
	// user goroutine
	{
		userproc, err := expproc.NewUsers(tmpdir)
		if err != nil {
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Users(ctx, userproc); err != nil {
				errC <- fmt.Errorf("error listing users: %w", err)
				return
			}
			if err := userproc.Close(); err != nil {
				errC <- fmt.Errorf("error closing user processor: %w", err)
				return
			}
			lg.Debug("users done")
		}()
	}
	// conversations goroutine
	{
		wg.Add(1)
		go func() {
			defer wg.Done()

			conv, err := expproc.NewConversation(tmpdir)
			if err != nil {
				errC <- err
				return
			}

			if err := s.AsyncConversations(ctx, conv, links, func(sr slackdump.StreamResult) error {
				if sr.IsLast {
					return conv.Finalise(sr.ChannelID)
				}
				lg.Printf("finished: %s", sr)
				return nil
			}); err != nil {
				errC <- fmt.Errorf("error streaming conversations: %w", err)
				return
			}
			lg.Debug("conversations done")
		}()
	}
	// sentinel
	go func() {
		wg.Wait()
		close(errC)
	}()

	lg.Printf("waiting for the conversations export to finish")
	// process returned errors
	for err := range errC {
		if err != nil {
			return err
		}
	}
	lg.Printf("conversations export finished")
	return nil
}
