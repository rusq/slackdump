package diag

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/logger"
	"golang.org/x/time/rate"
)

var cmdSearch = &base.Command{
	UsageLine:   "slackdump tools search",
	Short:       "searches for messages matching criteria",
	Long:        "Experimental command to search for messages matching criteria.",
	RequireAuth: true,
	Run:         runSearch,
	FlagMask:    cfg.OmitAll &^ cfg.OmitAuthFlags,
	PrintFlags:  true,
}

var searchFlags = struct {
	perPage uint
}{
	perPage: 50,
}

func init() {
	cmdSearch.Flag.UintVar(&searchFlags.perPage, "per-page", searchFlags.perPage, "number of messages per page")
}

func runSearch(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing query parameter")
	}
	prov, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}

	sd, err := slackdump.New(ctx, prov)
	if err != nil {
		return err
	}
	cl := sd.Client()

	query := args[0]

	lim := rate.NewLimiter(rate.Every(3*time.Second), 5)
	lg := logger.FromContext(ctx)
	var p = slack.SearchParameters{
		Sort:          "timestamp",
		SortDirection: "desc",
		Count:         int(searchFlags.perPage),
		Cursor:        "*",
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	for {
		sm, err := cl.SearchMessagesContext(ctx, query, p)
		if err != nil {
			return err
		}
		enc.Encode(sm.Matches)

		if sm.NextCursor == "" {
			lg.Print("no more messages")
			break
		}
		lg.Printf("cursor %s", sm.NextCursor)
		p.Cursor = sm.NextCursor

		lim.Wait(ctx)
	}

	return nil
}
