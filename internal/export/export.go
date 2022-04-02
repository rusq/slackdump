package export

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rusq/slackdump"
	"github.com/slack-go/slack"
)

type Export struct {
	dir string

	dumper *slackdump.SlackDumper
}

func New(dir string, dumper *slackdump.SlackDumper) *Export {
	return &Export{dir: dir, dumper: dumper}
}

func (se *Export) Run(ctx context.Context) error {
	// export users to users.json
	users, err := se.users(ctx)
	if err != nil {
		return err
	}

	// export channels to channels.json
	if err := se.channels(ctx, users); err != nil {
		return err
	}
	return nil
}

func (se *Export) users(ctx context.Context) (slackdump.Users, error) {
	// fetch users and save them.
	users, err := se.dumper.GetUsers(ctx)
	if err != nil {
		return nil, err
	}
	if err := se.serializeToFile(filepath.Join(se.dir, "users.json"), users); err != nil {
		return nil, err
	}

	return users, nil
}

func (se *Export) channels(ctx context.Context, users slackdump.Users) error {
	if err := se.dumper.StreamChannels(ctx, slackdump.AllChanTypes, func(ch slack.Channel) error {
		messages, err := se.dumper.DumpMessages(ctx, ch.ID)
		if err != nil {
			return fmt.Errorf("failed dumping %q (%s): %w", ch.Name, ch.ID, err)
		}
		se.byDate(messages, users)
		return nil
	}); err != nil {
		return fmt.Errorf("channels: error: %w", err)
	}
	return nil
}

// serialize writes the data in json format to provided filename.
func (se *Export) serializeToFile(filename string, data any) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("serializeToFile: failed to create %q: %w", filename, err)
	}
	defer f.Close()

	return se.serialize(f, data)
}

func (*Export) serialize(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("serialize: failed to encode: %w", err)
	}

	return nil
}
