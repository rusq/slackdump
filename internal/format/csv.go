package format

import (
	"context"
	"encoding/csv"
	"io"
	"strconv"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/types"
)

type CSV struct {
	opts options
}

type csvOptions struct {
	UseCRLF bool
	Comma   rune
}

func init() {
	Converters[CCSV] = NewCSV
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

	for _, u := range chans {
		if err := csv.Write([]string{
			u.ID,
			NVL(u.Name, ui.DisplayName(u.User)),
			_ft(int64(u.Created)),
			_fb(u.IsArchived),
			_fb(u.IsChannel),
			_fb(u.IsMpIM),
			_fb(u.IsPrivate),
			_fb(u.IsIM),
			u.Purpose.Value,
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
