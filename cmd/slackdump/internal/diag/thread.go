package diag

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/rusq/osenv/v2"
	"github.com/schollz/progressbar/v3"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
)

var _ = godotenv.Load()

var CmdThread = &base.Command{
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
	CmdThread.Run = runThread
	CmdThread.Flag.Usage = func() {
		fmt.Fprint(os.Stdout, "usage: slackdump diag thread [flags]\n\nFlags:\n")
		CmdThread.Flag.PrintDefaults()
	}
}

var (
	// TODO: test with client auth.
	token        = CmdThread.Flag.String("app-token", os.Getenv("APP_TOKEN"), "Slack application or bot token")
	channel      = CmdThread.Flag.String("channel", osenv.Value("CHANNEL", ""), "channel to operate on")
	numThreadMsg = CmdThread.Flag.Int("num", 2, "number of messages to generate in the thread")
	delThread    = CmdThread.Flag.String("del", "", "`URL` of the thread to delete")
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

	if *delThread != "" {
		if err := runDelete(*token, *delThread); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
	} else {
		if err := runGenerate(*token, *channel, *numThreadMsg); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
	}
	return nil
}

func runDelete(token, url string) error {
	if err := deleteThread(context.Background(), slack.New(token), url); err != nil {
		return err
	}
	return nil
}

func runGenerate(token string, channelID string, numMsg int) error {
	if channelID == "" {
		return errors.New("channel ID is required")
	}
	if err := generateThread(context.Background(), slack.New(token), channelID, numMsg); err != nil {
		return err
	}
	return nil
}

func generateThread(ctx context.Context, client *slack.Client, channelID string, numMsg int) error {
	msg := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", fmt.Sprintf("Very long thread (%d messages)", numMsg), true, false),
		),
	}
	_, ts, err := client.PostMessageContext(
		ctx,
		channelID,
		slack.MsgOptionBlocks(msg...),
	)
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	l := network.NewLimiter(network.Tier3, slackdump.DefLimits.Tier3.Burst, int(slackdump.DefLimits.Tier3.Boost))
	pb := progressbar.Default(int64(numMsg))
	pb.Describe("posting messages")
	defer pb.Finish()
	for i := 0; i < numMsg; i++ {
		if err := network.WithRetry(ctx, l, 3, func() error {
			_, _, err := client.PostMessageContext(ctx, channelID, slack.MsgOptionTS(ts), slack.MsgOptionText(fmt.Sprintf("message: %d", i), false))
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

func delMessages(ctx context.Context, client *slack.Client, channelID string, msgs []slack.Message) error {
	pb := progressbar.Default(int64(len(msgs)))
	pb.Describe("deleting messages")

	defer pb.Finish()

	l := network.NewLimiter(network.Tier3, slackdump.DefLimits.Tier3.Burst, int(slackdump.DefLimits.Tier3.Boost))
	for _, m := range msgs {
		err := network.WithRetry(ctx, l, 3, func() error {
			_, _, err := client.DeleteMessageContext(ctx, channelID, m.Timestamp)
			return err
		})
		if err != nil {
			return err
		}
		_ = pb.Add(1)
	}
	return nil
}

func getMessages(ctx context.Context, client *slack.Client, ui *structures.SlackLink) ([]slack.Message, error) {
	var msgs []slack.Message
	cursor := ""
	for {
		var (
			chunk   []slack.Message
			hasmore bool
			err     error
		)
		chunk, hasmore, cursor, err = client.GetConversationRepliesContext(
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
