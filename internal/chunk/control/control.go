package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
)

type Controller struct {
	erc EncodeReferenceCloser
	s   Streamer
	options
}

// New creates a new generic [Controller], that accepts
// [EncodeReferenceCloser]. Once the [Control.Close] is called it closes all
// processors, including the [EncodeReferenceCloser].
func New(ctx context.Context, s Streamer, erc EncodeReferenceCloser, opts ...Option) (*Controller, error) {
	d := &Controller{
		erc: erc,
		s:   s,
		options: options{
			lg:    slog.Default(),
			tf:    &noopTransformer{},
			filer: &processor.NopFiler{},
			avp:   &processor.NopAvatars{},
		},
	}
	for _, opt := range opts {
		opt(&d.options)
	}
	return d, nil
}

// newConvTransformer creates a new conversation transformer.
func (c *Controller) newConvTransformer(ctx context.Context) *conversationTransformer {
	return &conversationTransformer{
		ctx: ctx,
		tf:  c.tf,
		rc:  c.erc,
	}
}

// Run starts the scraping of the Slack API. The [EntityList] is used to
// determine which entities to scrape. The [EntityList] can be created with the
// [structures.NewEntityList] function.
func (c *Controller) Run(ctx context.Context, list *structures.EntityList) error {
	rec := chunk.NewCustomRecorder(c.erc)
	defer rec.Close()

	streamer, proc := c.mkSuperprocessor(ctx, rec)

	return runWorkers(ctx, streamer, list, proc, c.flags)
}

func (c *Controller) mkSuperprocessor(ctx context.Context, rec *chunk.Recorder) (Streamer, superprocessor) {
	streamer := c.s
	// got to do some explanation here: the order of processors is important:
	// files ==> recorder ==> transformer                     2     1            3
	conv := processor.AppendMessenger(processor.PrependFiler(rec, c.filer), c.newConvTransformer(ctx))
	if c.flags.ChannelUsers {
		// userIDCollector collects the user IDs from messages and thread messages (excluding duplicates)
		// and sends them to the userIDC channel.  The userCollectingStreamer replaces the Users method
		// of the Streamer with a method that gets the information for user IDs received on the userIDC
		// channel and calls the Users processor method.  Once the Close method is called on userIDCollector,
		// the userID channel is closed, and the userCollectingStreamer stops processing the user IDs.
		//
		// Drawback is that the transformer won't start until all user IDs are collected from all channels.
		ucoll := newUserIDCollector()
		conv = processor.PrependMessenger(conv, ucoll)
		streamer = &userCollectingStreamer{
			Streamer: streamer,
			userIDC:  ucoll.C(),
		}
	}

	sp := superprocessor{
		Conversations: conv,
		//                                       1                     2    3
		Users:         processor.JoinUsers(c.newUserCollector(ctx), c.avp, rec),
		Channels:      rec,
		WorkspaceInfo: rec,
	}

	return streamer, sp
}

type SearchType int

const (
	SMessages SearchType = 1 << iota
	SFiles

	srchUnknown SearchType = 0
)

// Search starts the search for the query string. The search type is defined by
// the [SearchType] parameter. The search is done in parallel for messages and
// files.
func (c *Controller) Search(ctx context.Context, query string, stype SearchType) error {
	rec := chunk.NewCustomRecorder(c.erc)
	defer rec.Close()

	p := &jointFileSearcher{
		FileSearcher: rec,
		filer:        processor.JoinFilers(rec, c.filer), // in case we have a downloader, we need to join it
	}

	s := supersearcher{
		WorkspaceInfo:   rec,
		MessageSearcher: rec,
		FileSearcher:    p,
	}

	start := time.Now()
	if err := runSearch(ctx, c.s, s, stype, query); err != nil {
		return fmt.Errorf("error searching: %w", err)
	}
	c.lg.InfoContext(ctx, "search completed", "query", query, "took", time.Since(start).String())
	return nil
}

// Close closes the controller and all its file processors.
func (c *Controller) Close() error {
	var errs error
	if c.filer != nil {
		if err := c.filer.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("error closing file processor: %w", err))
		}
	}
	if c.avp != nil {
		if err := c.avp.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("error closing avatar processor: %w", err))
		}
	}
	// TODO: Decide if it is necessary to close the encoder here or leave it
	// for the caller.  Maybe make it conditional?
	if err := c.erc.Close(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("error closing database processor: %w", err))
	}
	return errs
}
