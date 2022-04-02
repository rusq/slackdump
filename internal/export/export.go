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
		return se.exportConversation(ctx, ch, users)
	}); err != nil {
		return fmt.Errorf("channels: error: %w", err)
	}
	return nil
}

func (se *Export) exportConversation(ctx context.Context, ch slack.Channel, users slackdump.Users) error {
	messages, err := se.dumper.DumpMessages(ctx, ch.ID)
	if err != nil {
		return fmt.Errorf("failed dumping %q (%s): %w", ch.Name, ch.ID, err)
	}

	msgs, err := se.byDate(messages, users)
	if err != nil {
		return fmt.Errorf("exportChannelData: error: %w", err)
	}

	if err := se.saveChannel(ch.Name, msgs); err != nil {
		return err
	}

	return nil
}

// saveChannel creates a directory `name` and writes the contents of msgs. for
// each map key the json file is created, with the name `{key}.json`, and values
// for that key are serialised to the file in json format.
func (se *Export) saveChannel(channelName string, msgs messagesByDate) error {
	basedir := filepath.Join(se.dir, channelName)
	if err := os.MkdirAll(basedir, 0700); err != nil {
		return fmt.Errorf("unable to create directory %q: %w", channelName, err)
	}
	for date, messages := range msgs {
		output := filepath.Join(basedir, date+".json")
		if err := se.serializeToFile(output, messages); err != nil {
			return err
		}
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
