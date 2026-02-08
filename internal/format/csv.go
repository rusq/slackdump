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
package format

import (
	"context"
	"encoding/csv"
	"io"
	"strconv"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/types"
)

type CSV struct {
	opts options
}

type csvOptions struct {
	UseCRLF bool
	Comma   rune
}

func init() {
	converters[CCSV] = NewCSV
}

func NewCSV(opts ...Option) Formatter {
	settings := options{
		csvOptions: csvOptions{
			UseCRLF: false,
			Comma:   ',',
		},
	}
	for _, fn := range opts {
		fn(&settings)
	}
	return &CSV{settings}
}

// Extension returns the file extension for the formatter.
func (c CSV) Extension() string {
	return ".csv"
}

// timestamp, channel, username, text

func (c *CSV) Conversation(ctx context.Context, w io.Writer, u []slack.User, conv *types.Conversation) error {
	csv := c.mkwriter(w)
	defer csv.Flush()

	ui := types.Users(u).IndexByID()
	repl := userReplacer(ui)

	for _, m := range conv.Messages {
		if err := csv.Write([]string{m.Timestamp, conv.Name, ui.DisplayName(m.User), repl.Replace(m.Text)}); err != nil {
			return err
		}
	}
	return nil
}

var (
	// formatting functions
	_fb = strconv.FormatBool
	_ft = func(sec int64) string {
		t := time.Unix(sec, 0)
		return t.Format("2006-01-02 15:04:05")
	}
)

func (c *CSV) Channels(ctx context.Context, w io.Writer, u []slack.User, chans []slack.Channel) error {
	csv := c.mkwriter(w)
	defer csv.Flush()

	if c.opts.bare {
		return c.channelsBare(ctx, csv, u, chans)
	} else {
		return c.channelsFull(ctx, csv, u, chans)
	}
}

func (c *CSV) channelsBare(_ context.Context, csv *csv.Writer, _ []slack.User, chans []slack.Channel) error {
	for _, c := range chans {
		if err := csv.Write([]string{c.ID}); err != nil {
			return err
		}
	}
	return nil
}

func (c *CSV) channelsFull(_ context.Context, csv *csv.Writer, u []slack.User, chans []slack.Channel) error {
	if err := csv.Write([]string{
		"ID",
		"Name",
		"Created",
		"Is Archived?",
		"Is Channel?",
		"Is MPIM?",
		"Is Private?",
		"Is IM?",
		"Purpose",
	}); err != nil {
		return err
	}

	ui := types.Users(u).IndexByID()

	for _, c := range chans {
		if err := csv.Write([]string{
			c.ID,
			NVL(c.Name, ui.DisplayName(c.User)),
			_ft(int64(c.Created)),
			_fb(c.IsArchived),
			_fb(c.IsChannel),
			_fb(c.IsMpIM),
			_fb(c.IsPrivate),
			_fb(c.IsIM),
			c.Purpose.Value,
		}); err != nil {
			return err
		}
	}
	return nil
}

func NVL(s string, rest ...string) string {
	if s != "" {
		return s
	}
	for _, s = range rest {
		if s != "" {
			return s
		}
	}
	return ""
}

func (c *CSV) Users(ctx context.Context, w io.Writer, users []slack.User) error {
	csv := c.mkwriter(w)
	defer csv.Flush()

	if c.opts.bare {
		return c.usersBare(ctx, csv, users)
	} else {
		return c.usersFull(ctx, csv, users)
	}
}

func (c *CSV) usersBare(_ context.Context, csv *csv.Writer, users []slack.User) error {
	for _, u := range users {
		if err := csv.Write([]string{u.ID}); err != nil {
			return err
		}
	}
	return nil
}

func (c *CSV) usersFull(_ context.Context, csv *csv.Writer, users []slack.User) error {
	if err := csv.Write([]string{
		"ID",
		"Team ID",
		"Name",
		"Is Admin?",
		"Last Updated",
		"Is Deleted?",
		"Is Bot?",
		"Real Name",
		"Email",
		"Title",
		"Timezone",
	}); err != nil {
		return err
	}

	fb := strconv.FormatBool

	for _, u := range users {
		if err := csv.Write([]string{
			u.ID,
			u.TeamID,
			u.Name,
			fb(u.IsAdmin),
			u.Updated.Time().Format("2006-01-02 15:04:05"),
			fb(u.Deleted),
			fb(u.IsBot),
			u.RealName,
			u.Profile.Email,
			u.Profile.Title,
			u.TZ,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *CSV) mkwriter(w io.Writer) *csv.Writer {
	csv := csv.NewWriter(w)
	csv.Comma = c.opts.Comma
	csv.UseCRLF = c.opts.UseCRLF
	return csv
}
