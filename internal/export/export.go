package export

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"

	"github.com/rusq/slackdump"
	"github.com/slack-go/slack"
)

const userPrefix = "IM-" // prefix for Direct Messages

type Export struct {
	dir    string                 //target directory
	dumper *slackdump.SlackDumper // slackdumper instance
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

	var chans []slack.Channel
	if err := se.dumper.StreamChannels(ctx, slackdump.AllChanTypes, func(ch slack.Channel) error {
		if err := se.exportConversation(ctx, ch, users); err != nil {
			return err
		}

		chans = append(chans, ch)

		return nil

	}); err != nil {
		return fmt.Errorf("channels: error: %w", err)
	}

	se.serializeToFile(filepath.Join(se.dir, "channels.json"), chans)

	return nil
}

func (se *Export) exportConversation(ctx context.Context, ch slack.Channel, users slackdump.Users) error {
	messages, err := se.dumper.DumpAllMessages(ctx, ch.ID)
	if err != nil {
		return fmt.Errorf("failed dumping %q (%s): %w", ch.Name, ch.ID, err)
	}

	msgs, err := se.byDate(messages, users)
	if err != nil {
		return fmt.Errorf("exportChannelData: error: %w", err)
	}

	name, err := validName(ctx, ch, users.IndexByID())
	if err != nil {
		return err
	}

	if err := se.saveChannel(name, msgs); err != nil {
		return err
	}

	return nil
}

var errUnknownEntity = errors.New("encountered an unknown entity, please (1) rerun with -trace=trace.out, (2) create an issue on https://github.com/rusq/slackdump/issues and (3) submit the trace file when requested")

// validName returns the channel or user name.  If it is not able to determine
// either of those, it will return the ID of the channel or a user.
//
// I have no access to Enterprise Plan Slack Export functionality, so I don't
// know what directory name would IM have.  So, I'll do the right thing, and
// prefix IM directories with `userPrefix`.
//
// If it fails to determine the appropriate name, it returns errUnknownEntity.
func validName(ctx context.Context, ch slack.Channel, uidx userIndex) (string, error) {
	if ch.NameNormalized != "" {
		// populated on all channels, private channels, and group messages
		return ch.NameNormalized, nil
	}

	// user branch

	if !ch.IsIM {
		// what is this? It doesn't have a name, and is not a IM.
		trace.Logf(ctx, "unsupported", "unknown type=%s", traceCompress(ctx, ch))
		return "", errUnknownEntity
	}
	user, ok := uidx[ch.User]
	if ok {
		return userPrefix + user.Name, nil
	}

	// failed to get the username

	trace.Logf(ctx, "warning", "user not found: %s", ch.User)

	// using ID as a username
	return userPrefix + ch.User, nil
}

// traceCompress gz-compresses and base64-encodes the json data for trace.
func traceCompress(ctx context.Context, v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		trace.Logf(ctx, "error", "error marshalling v: %s", err)
		return ""
	}
	var buf bytes.Buffer
	b64 := base64.NewEncoder(base64.RawStdEncoding, &buf)
	gz := gzip.NewWriter(b64)
	if _, err := gz.Write(data); err != nil {
		trace.Logf(ctx, "error", "error compressing data: %v", err)
	}
	gz.Close()
	b64.Close()
	return buf.String()
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
