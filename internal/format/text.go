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
	"bufio"
	"context"
	"fmt"
	"html"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/types"
)

var _ Formatter = &Text{}

const (
	defaultMsgSplitAfter = 3 * time.Minute
	textTimeFmt          = "02/01/2006 15:04:05 Z0700"
)

type Text struct {
	opts options
}

type textOptions struct {
	msgSplitAfter time.Duration
}

func TextNewMessageThreshold(d time.Duration) Option {
	return func(o *options) {
		o.textOptions.msgSplitAfter = d
	}
}

func init() {
	converters[CText] = NewText
}

func NewText(opts ...Option) Formatter {
	settings := options{
		textOptions: textOptions{
			msgSplitAfter: defaultMsgSplitAfter,
		},
	}
	for _, fn := range opts {
		fn(&settings)
	}
	return &Text{opts: settings}
}

// Extension returns the file extension for the formatter.
func (txt Text) Extension() string {
	return ".txt"
}

func (txt *Text) Conversation(ctx context.Context, w io.Writer, u []slack.User, conv *types.Conversation) error {
	buf := bufio.NewWriter(w)
	defer buf.Flush()

	ui := structures.NewUserIndex(u)

	return txt.txtConversations(w, conv.Messages, "", ui, userReplacer(ui))
}

func (txt *Text) txtConversations(w io.Writer, m []types.Message, prefix string, userIdx structures.UserIndex, repl *strings.Replacer) error {
	var (
		prevMsg  types.Message
		prevTime time.Time
	)
	for _, message := range m {
		t, err := structures.ParseSlackTS(message.Timestamp)
		if err != nil {
			return err
		}
		diff := t.Sub(prevTime)
		if prevMsg.User == message.User && diff < txt.opts.msgSplitAfter {
			fmt.Fprintf(w, prefix+"%s\n", message.Text)
		} else {
			fmt.Fprintf(w, prefix+"\n"+prefix+"> %s [%s] @ %s:\n%s\n",
				userIdx.Sender(&message.Message), message.User,
				t.Format(textTimeFmt),
				prefix+html.UnescapeString(repl.Replace(message.Text)),
			)
		}
		if len(message.ThreadReplies) > 0 {
			if err := txt.txtConversations(w, message.ThreadReplies, "|   ", userIdx, repl); err != nil {
				return err
			}
		}
		prevMsg = message
		prevTime = t
	}
	return nil
}

func (txt *Text) Users(ctx context.Context, w io.Writer, u []slack.User) error {
	if txt.opts.bare {
		return txt.usersBare(ctx, w, u)
	}
	return txt.usersFull(ctx, w, u)
}

func (txt *Text) usersBare(_ context.Context, w io.Writer, u []slack.User) error {
	for i := range u {
		fmt.Fprintf(w, "%s\n", u[i].ID)
	}
	return nil
}

func (txt *Text) usersFull(_ context.Context, w io.Writer, u []slack.User) error {
	const strFormat = "%s\t%s\t%s\t%s\t%s\t%s\n"
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer writer.Flush()

	// header
	if _, err := fmt.Fprintf(writer, strFormat, "Name", "ID", "Bot?", "Email", "Deleted?", "Restricted?"); err != nil {
		return fmt.Errorf("writer error: %w", err)
	}
	if _, err := fmt.Fprintf(writer, strFormat, "", "", "", "", "", ""); err != nil {
		return fmt.Errorf("writer error: %w", err)
	}

	var (
		names   = make([]string, 0, len(u))
		usermap = make(structures.UserIndex, len(u))
	)
	for i := range u {
		names = append(names, u[i].Name)
		usermap[u[i].Name] = &u[i]
	}
	sort.Strings(names)

	// data
	for _, name := range names {
		var (
			deleted    string
			bot        string
			restricted string
		)
		if usermap[name].Deleted {
			deleted = "deleted"
		}
		if usermap[name].IsBot {
			bot = "bot"
		}
		if usermap[name].IsRestricted {
			restricted = "restricted"
		}

		_, err := fmt.Fprintf(writer, strFormat,
			name, usermap[name].ID, bot, usermap[name].Profile.Email, deleted, restricted,
		)
		if err != nil {
			return fmt.Errorf("writer error: %w", err)
		}
	}
	return nil
}

func (txt *Text) Channels(ctx context.Context, w io.Writer, u []slack.User, cc []slack.Channel) error {
	if txt.opts.bare {
		return txt.channelsBare(ctx, w, u, cc)
	}
	return txt.channelsFull(ctx, w, u, cc)
}

func (txt *Text) channelsBare(_ context.Context, w io.Writer, _ []slack.User, cc []slack.Channel) error {
	for i := range cc {
		fmt.Fprintf(w, "%s\n", cc[i].ID)
	}
	return nil
}

func (txt *Text) channelsFull(_ context.Context, w io.Writer, u []slack.User, cc []slack.Channel) error {
	const strFormat = "%s\t%s\t%s\n"

	ui := structures.NewUserIndex(u)

	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer writer.Flush()

	fmt.Fprintf(writer, strFormat, "ID", "Arch", "What")
	for i, ch := range cc {
		who := ui.ChannelName(ch)
		archived := "-"
		if cc[i].IsArchived || ui.IsDeleted(ch.User) {
			archived = "arch"
		}
		fmt.Fprintf(writer, strFormat, ch.ID, archived, who)
	}
	return nil
}
