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
	"path/filepath"
	"runtime/trace"
	"time"

	"github.com/rusq/dlog"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/fsadapter"
)

const userPrefix = "IM-" // prefix for Direct Messages

// Export is the instance of Slack Exporter.
type Export struct {
	fs     fsadapter.FS           // target filesystem
	dumper *slackdump.SlackDumper // slackdumper instance

	// time window
	opts Options
}

type Options struct {
	Oldest       time.Time
	Latest       time.Time
	IncludeFiles bool
}

func New(dumper *slackdump.SlackDumper, fs fsadapter.FS, cfg Options) *Export {
	return &Export{fs: fs, dumper: dumper, opts: cfg}
}

// Run runs the export.
func (se *Export) Run(ctx context.Context) error {
	// export users to users.json
	users, err := se.users(ctx)
	if err != nil {
		return err
	}

	// export channels to channels.json
	if err := se.messages(ctx, users); err != nil {
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
	if err := serializeToFS(se.fs, "users.json", users); err != nil {
		return nil, err
	}

	return users, nil
}

func (se *Export) messages(ctx context.Context, users slackdump.Users) error {
	var chans []slack.Channel
	dl := downloader.New(se.dumper.Client(), se.fs)
	if se.opts.IncludeFiles {
		// start the downloader
		dl.Start(ctx)
	}

	if err := se.dumper.StreamChannels(ctx, slackdump.AllChanTypes, func(ch slack.Channel) error {
		if err := se.exportConversation(ctx, ch, users, dl); err != nil {
			return err
		}

		chans = append(chans, ch)

		return nil

	}); err != nil {
		return fmt.Errorf("channels: error: %w", err)
	}

	populateNames(chans, users)

	return serializeToFS(se.fs, "channels.json", chans)
}

// populateNames updates name and name_normalized of the private conversations
// in ch with their respective usernames (see
// https://github.com/rusq/slackdump/issues/63 )
func populateNames(ch []slack.Channel, usr []slack.User) {
	// quick and dirty
	for i := range ch {
		if !ch[i].IsIM || ch[i].User == "" {
			// we don't care about public conversations in this case.
			continue
		}
		for k := range usr {
			if ch[i].User == usr[k].ID {
				ch[i].Name = usr[k].Name
				ch[i].NameNormalized = usr[k].Name
			}
		}
	}
}

func (se *Export) exportConversation(ctx context.Context, ch slack.Channel, users slackdump.Users, dl *downloader.Client) error {
	dlFn := se.downloadFn(dl, ch.Name)
	messages, err := se.dumper.DumpMessagesRaw(ctx, ch.ID, se.opts.Oldest, se.opts.Latest, dlFn)
	if err != nil {
		return fmt.Errorf("failed dumping %q (%s): %w", ch.Name, ch.ID, err)
	}
	if len(messages.Messages) == 0 {
		// empty result set
		return nil
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

// downloadFn returns the process function that should be passed to
// DumpMessagesRaw that will handle the download of the files.  If the
// downloader is not started, i.e. if file download is disabled, it will
// silently ignore the error and return nil.
func (se *Export) downloadFn(dl *downloader.Client, channelName string) func(msg []slackdump.Message, channelID string) (slackdump.ProcessResult, error) {
	dir := filepath.Join(se.basedir(channelName), "attachments")
	return func(msg []slackdump.Message, channelID string) (slackdump.ProcessResult, error) {
		files := se.dumper.ExtractFiles(msg)
		for _, f := range files {
			if err := dl.DownloadFile(dir, f); err != nil {
				if errors.Is(err, downloader.ErrNotStarted) {
					return slackdump.ProcessResult{Entity: "files", Count: 0}, nil
				}
				return slackdump.ProcessResult{}, err
			}
			dlog.Printf("sent: %s", f.Name)
		}
		return slackdump.ProcessResult{Entity: "files", Count: len(files)}, nil
	}
}

var errUnknownEntity = errors.New("encountered an unknown entity, please (1) rerun with -trace=trace.out, (2) create an issue on https://github.com/rusq/slackdump/issues and (3) submit the trace file when requested")

// validName returns the channel or user name.  If it is not able to determine
// either of those, it will return the ID of the channel or a user.
//
// I have no access to Enterprise Plan Slack Export functionality, so I don't
// know what directory name would IM have in Slack Export ZIP.  So, I'll do the
// right thing, and prefix IM directories with `userPrefix`.
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

func (se *Export) basedir(channelName string) string {
	return channelName
}

// saveChannel creates a directory `name` and writes the contents of msgs. for
// each map key the json file is created, with the name `{key}.json`, and values
// for that key are serialised to the file in json format.
func (se *Export) saveChannel(channelName string, msgs messagesByDate) error {
	for date, messages := range msgs {
		output := filepath.Join(channelName, date+".json")
		if err := serializeToFS(se.fs, output, messages); err != nil {
			return err
		}
	}
	return nil
}

// serializeToFS writes the data in json format to provided filesystem adapter.
func serializeToFS(fs fsadapter.FS, filename string, data any) error {
	f, err := fs.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return serialize(f, data)
}

func serialize(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("serialize: failed to encode: %w", err)
	}

	return nil
}
