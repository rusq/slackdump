package export

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/export/expproc"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/slack-go/slack"
)

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
