package processor

import (
	"context"
	"runtime"

	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
)

type Printer struct{}

func (d *Printer) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, messages []slack.Message) error {
	dlog.Printf("Discarding %d messages", len(messages))
	for i := range messages {
		dlog.Printf("  message: %s", messages[i].Timestamp)
	}
	return nil
}

func (d *Printer) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly, isLast bool, replies []slack.Message) error {
	dlog.Printf("Discarding %d replies to %s", len(replies), parent.Timestamp)
	for i := range replies {
		dlog.Printf("  reply: %s", replies[i].Timestamp)
	}
	return nil
}

func (d *Printer) Files(parent slack.Message, isThread bool, files []slack.File) error {
	dlog.Printf("Discarding %d files to %s (thread: %v)", len(files), parent.Timestamp, isThread)
	if parent.Timestamp == "" {
		runtime.Breakpoint()
	}
	for i := range files {
		dlog.Printf("  file: %s", files[i].ID)
	}
	return nil
}

func (d *Printer) Close() error {
	dlog.Println("Discarder closing")
	return nil
}
