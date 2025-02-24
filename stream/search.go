package stream

import (
	"context"
	"log/slog"
	"runtime/trace"
	"sync"

	"github.com/rusq/slack"
	"golang.org/x/sync/errgroup"

	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/rusq/slackdump/v3/processor"
)

// SearchMessages executes the search query and calls the processor for each
// message results, it will also collect information about the channels.
// Message search results do not have files attached, so do not expect Files
// method to be called.
func (cs *Stream) SearchMessages(ctx context.Context, proc processor.MessageSearcher, query string) error {
	ctx, task := trace.NewTask(ctx, "SearchMessages")
	defer task.End()

	var (
		srC          = make(chan Result, 1)
		channelInfoC = make(chan string, 100)
		// channelUsersC = make(chan string, 200)

		wg sync.WaitGroup
	)
	{
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(channelInfoC)
			// defer close(channelUsersC)

			if err := cs.searchmsg(ctx, query, func(sm []slack.SearchMessage) error {
				if err := proc.SearchMessages(ctx, query, sm); err != nil {
					return err
				}
				for _, m := range sm {
					select {
					case <-ctx.Done():
						return context.Cause(ctx)
						// collect channel ids
					case channelInfoC <- m.Channel.ID:
						// channelUsersC <- m.Channel.ID
					}
				}
				for _, fn := range cs.resultFn {
					if err := fn(Result{Type: RTSearch, Count: len(sm)}); err != nil {
						return err
					}
				}

				return nil
			}); err != nil {
				srC <- Result{Type: RTMain, Err: err}
			}
		}()
	}
	{
		wg.Add(1)
		go func() {
			cs.channelInfoWorker(ctx, proc, srC, channelInfoC)
			wg.Done()
		}()
	}
	// {
	// 	wg.Add(1)
	// 	go func() {
	// 		cs.channelUsersWorker(ctx, proc, srC, channelUsersC)
	// 		wg.Done()
	// 	}()
	// }
	go func() {
		wg.Wait()
		close(srC)
	}()
	for res := range srC {
		if err := res.Err; err != nil {
			return err
		}
	}
	return nil
}

func (cs *Stream) searchmsg(ctx context.Context, query string, fn func(sm []slack.SearchMessage) error) error {
	ctx, task := trace.NewTask(ctx, "searchMessages")
	defer task.End()

	lg := slog.With("query", query)

	p := slack.SearchParameters{
		Sort:          "timestamp",
		SortDirection: "desc",
		Count:         100,
		Cursor:        "*",
	}
	for {
		var (
			sm  *slack.SearchMessages
			err error
		)
		if err := network.WithRetry(ctx, cs.limits.searchmsg, cs.limits.tier.Tier2.Retries, func() error {
			sm, err = cs.client.SearchMessagesContext(ctx, query, p)
			return err
		}); err != nil {
			return err
		}
		if err := fn(sm.Matches); err != nil {
			return err
		}
		if sm.NextCursor == "" {
			lg.DebugContext(ctx, "SearchMessages:  no more messages")
			break
		}
		p.Cursor = sm.NextCursor
	}

	return nil
}

// SearchFiles executes the search query and calls the processor for each
// returned slice of files.  Channels do not have the file information.
func (cs *Stream) SearchFiles(ctx context.Context, proc processor.FileSearcher, query string) error {
	ctx, task := trace.NewTask(ctx, "SearchFiles")
	defer task.End()

	lg := slog.With("query", query)

	p := slack.SearchParameters{
		Sort:          "timestamp",
		SortDirection: "desc",
		Count:         100,
		Cursor:        "*",
	}
	for {
		var (
			sm  *slack.SearchFiles
			err error
		)
		if err := network.WithRetry(ctx, cs.limits.searchmsg, cs.limits.tier.Tier2.Retries, func() error {
			sm, err = cs.client.SearchFilesContext(ctx, query, p)
			return err
		}); err != nil {
			return err
		}
		if err := proc.SearchFiles(ctx, query, sm.Matches); err != nil {
			return err
		}
		if err := proc.Files(ctx, &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "SEARCH"}}}, slack.Message{}, sm.Matches); err != nil {
			return err
		}
		if sm.NextCursor == "" {
			lg.DebugContext(ctx, "SearchFiles:  no more messages")
			break
		}
		p.Cursor = sm.NextCursor
	}

	return nil
}

func (s *Stream) Search(ctx context.Context, proc processor.Searcher, query string) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return s.SearchMessages(ctx, proc, query)
	})
	eg.Go(func() error {
		return s.SearchFiles(ctx, proc, query)
	})

	return eg.Wait()
}
