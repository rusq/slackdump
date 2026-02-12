// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package diag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rusq/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/chunk"
	dirproc "github.com/rusq/slackdump/v4/internal/chunk/backend/directory"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/types"
)

var cmdSearch = &base.Command{
	UsageLine: "slackdump tools search",
	Short:     "searches for messages matching criteria",
	Long:      "Experimental command to search for messages matching criteria.",
	Commands: []*base.Command{
		cmdSearchRun,
		cmdSearchConvert,
	},
}

var cmdSearchRun = &base.Command{
	UsageLine:   "slackdump tools search query [flags]",
	Short:       "searches for messages matching criteria",
	Long:        "Experimental command to search for messages matching criteria.",
	RequireAuth: true,
	Run:         runSearch,
	FlagMask:    cfg.OmitAll &^ cfg.OmitAuthFlags,
	PrintFlags:  true,
}

var searchFlags = struct {
	perPage  uint
	convert  bool
	channels string
	users    string
}{
	perPage:  100,
	convert:  false,
	channels: "",
	users:    "",
}

func init() {
	cmdSearch.Flag.UintVar(&searchFlags.perPage, "per-page", searchFlags.perPage, "number of messages per page")
}

func runSearch(ctx context.Context, cmd *base.Command, args []string) error {
	if searchFlags.convert {
		return runSearchConvert(ctx, cmd, args)
	}
	if len(args) < 1 {
		return fmt.Errorf("missing query parameter")
	}
	prov, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}

	hcl, err := prov.HTTPClient()
	if err != nil {
		return err
	}
	cl := slack.New(prov.SlackToken(), slack.OptionHTTPClient(hcl))

	query := args[0]

	lim := rate.NewLimiter(rate.Every(3*time.Second), 5)
	lg := cfg.Log
	p := slack.SearchParameters{
		Sort:          "timestamp",
		SortDirection: "desc",
		Count:         int(searchFlags.perPage),
		Cursor:        "*",
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	for {
		var (
			sm  *slack.SearchMessages
			err error
		)
		if err := network.WithRetry(ctx, lim, 3, func(ctx context.Context) error {
			sm, err = cl.SearchMessagesContext(ctx, query, p)
			return err
		}); err != nil {
			return err
		}
		enc.Encode(sm.Matches)

		if sm.NextCursor == "" {
			lg.Info("no more messages")
			break
		}
		lg.Info("paginating", "cursor", sm.NextCursor)
		p.Cursor = sm.NextCursor

		if err := lim.Wait(ctx); err != nil {
			return err
		}
	}

	return nil
}

var cmdSearchConvert = &base.Command{
	UsageLine:   "slackdump tools search convert",
	Short:       "converts experimental search output to chunks",
	Long:        "Convert results of the experimental search to chunks",
	RequireAuth: false,
	Run:         runSearchConvert,
	FlagMask:    cfg.OmitAll &^ cfg.OmitOutputFlag,
	PrintFlags:  true,
}

func init() {
	cmdSearchConvert.Flag.StringVar(&searchFlags.channels, "channels", searchFlags.channels, "channels file produced by list channels")
	cmdSearchConvert.Flag.StringVar(&searchFlags.users, "users", searchFlags.users, "users file produced by list users")
}

func runSearchConvert(ctx context.Context, _ *base.Command, args []string) error {
	var r io.ReadCloser
	if len(args) == 0 {
		r = os.Stdin
	} else {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	cfg.Output = cfg.StripZipExt(cfg.Output)
	if cfg.Output == "" {
		return errors.New("output is empty")
	}

	if err := os.MkdirAll(cfg.Output, 0o755); err != nil {
		return err
	}

	cd, err := chunk.OpenDir(cfg.Output)
	if err != nil {
		return err
	}
	defer cd.Close()

	dps, err := dirproc.NewSearch(cd, nil)
	if err != nil {
		return err
	}
	defer dps.Close()

	chans := make(map[string]struct{})
	dec := json.NewDecoder(r)
	for {
		var sm []slack.SearchMessage
		if err := dec.Decode(&sm); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		for _, m := range sm {
			chans[m.Channel.ID] = struct{}{}
		}
		if err := dps.SearchMessages(ctx, "query", sm); err != nil {
			return err
		}
	}

	if searchFlags.channels != "" {
		if err := convertChannels(ctx, dps, searchFlags.channels, chans); err != nil {
			return err
		}
	}
	if searchFlags.users != "" {
		dpu, err := dirproc.NewUsers(cd)
		if err != nil {
			return err
		}
		defer dpu.Close()
		if err := convertUsers(ctx, dpu, searchFlags.users); err != nil {
			return err
		}
	}
	return nil
}

func convertChannels(ctx context.Context, dps *dirproc.Search, filename string, chans map[string]struct{}) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	var ch types.Channels
	if err := json.NewDecoder(f).Decode(&ch); err != nil {
		return err
	}
	for _, c := range ch {
		if _, found := chans[c.ID]; found {
			if err := dps.ChannelInfo(ctx, &c, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

func convertUsers(ctx context.Context, dpu *dirproc.Users, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	var uu types.Users
	if err := json.NewDecoder(f).Decode(&uu); err != nil {
		return err
	}
	return dpu.Users(ctx, uu)
}
