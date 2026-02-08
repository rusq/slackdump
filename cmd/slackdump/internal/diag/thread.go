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
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/rusq/osenv/v2"
	"github.com/rusq/slack"
	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/workspace"
	"github.com/rusq/slackdump/v4/internal/client"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/structures"
)

var _ = godotenv.Load()

var cmdThread = &base.Command{
	Run:       nil,
	UsageLine: "slackdump tools thread [flags]",
	Short:     "thread utilities",
	Long: `
Thread is an utility that provides some useful functions for
testing, i.e. deletion of the threads, or generation of large threads.
`,
	FlagMask:    cfg.OmitAll,
	PrintFlags:  true,
	CustomFlags: true,
}

func init() {
	cmdThread.Run = runThread
	cmdThread.Flag.Usage = func() {
		fmt.Fprint(os.Stdout, "usage: slackdump tools thread [flags]\n\nFlags:\n")
		cmdThread.Flag.PrintDefaults()
	}
}

var (
	// TODO: test with client auth.
	channel      = cmdThread.Flag.String("channel", osenv.Value("CHANNEL", ""), "channel to operate on")
	numThreadMsg = cmdThread.Flag.Int("num", 2, "number of messages to generate in the thread")
	delThread    = cmdThread.Flag.String("del", "", "`URL` of the thread to delete")
)

func runThread(ctx context.Context, cmd *base.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return nil
	}

	if *channel == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("-channel flag is required")
	}

	ctx, err := workspace.CurrentOrNewProviderCtx(ctx)
	if err != nil {
		return err
	}
	prov, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}
	client, err := client.New(ctx, prov)
	if err != nil {
		return err
	}
	scl, ok := client.Client()
	if !ok {
		return errors.New("failed to get slack client")
	}

	if *delThread != "" {
		if err := runDelete(ctx, scl, *delThread); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
	} else {
		if err := runGenerate(ctx, scl, *channel, *numThreadMsg); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
	}
	return nil
}

func runDelete(ctx context.Context, cl *slack.Client, url string) error {
	if err := deleteThread(ctx, cl, url); err != nil {
		return err
	}
	return nil
}

func runGenerate(ctx context.Context, cl *slack.Client, channelID string, numMsg int) error {
	if channelID == "" {
		return errors.New("channel ID is required")
	}
	if err := generateThread(ctx, cl, channelID, numMsg); err != nil {
		return err
	}
	return nil
}

func generateThread(ctx context.Context, cl *slack.Client, channelID string, numMsg int) error {
	msg := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", fmt.Sprintf("Very long thread (%d messages)", numMsg), true, false),
		),
	}
	_, ts, err := cl.PostMessageContext(
		ctx,
		channelID,
		slack.MsgOptionBlocks(msg...),
	)
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	l := network.NewLimiter(network.Tier3, network.DefLimits.Tier3.Burst, int(network.DefLimits.Tier3.Boost))
	pb := progressbar.Default(int64(numMsg))
	pb.Describe("posting messages")
	defer func() { _ = pb.Finish() }()
	for i := 0; i < numMsg; i++ {
		if err := network.WithRetry(ctx, l, 3, func(ctx context.Context) error {
			_, _, err := cl.PostMessageContext(ctx, channelID, slack.MsgOptionTS(ts), slack.MsgOptionText(fmt.Sprintf("message: %d", i), false))
			return err
		}); err != nil {
			return fmt.Errorf("failed to post message to the thread: %w", err)
		}
		if err := pb.Add(1); err != nil {
			// what a shame
			return err
		}
	}
	return nil
}

func deleteThread(ctx context.Context, client *slack.Client, url string) error {
	ui, err := structures.ParseURL(url)
	if err != nil {
		return err
	}
	msgs, err := getMessages(ctx, client, ui)
	if err != nil {
		return err
	}
	if err := delMessages(ctx, client, ui.Channel, msgs); err != nil {
		return err
	}

	return nil
}

func delMessages(ctx context.Context, cl *slack.Client, channelID string, msgs []slack.Message) error {
	pb := progressbar.Default(int64(len(msgs)))
	pb.Describe("deleting messages")

	defer func() { _ = pb.Finish() }()

	l := network.NewLimiter(network.Tier3, network.DefLimits.Tier3.Burst, int(network.DefLimits.Tier3.Boost))
	for _, m := range msgs {
		err := network.WithRetry(ctx, l, 3, func(ctx context.Context) error {
			_, _, err := cl.DeleteMessageContext(ctx, channelID, m.Timestamp)
			return err
		})
		if err != nil {
			return err
		}
		_ = pb.Add(1)
	}
	return nil
}

func getMessages(ctx context.Context, cl *slack.Client, ui *structures.SlackLink) ([]slack.Message, error) {
	var msgs []slack.Message
	cursor := ""
	for {
		var (
			chunk   []slack.Message
			hasmore bool
			err     error
		)
		chunk, hasmore, cursor, err = cl.GetConversationRepliesContext(
			ctx,
			&slack.GetConversationRepliesParameters{
				ChannelID: ui.Channel,
				Timestamp: ui.ThreadTS,
				Cursor:    cursor,
			})
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, chunk...)
		if !hasmore {
			break
		}
	}
	return msgs, nil
}
