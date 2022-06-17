package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"time"

	"github.com/rusq/dlog"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/rusq/slackdump/v2/types"
)

// Export is the instance of Slack Exporter.
type Export struct {
	fs fsadapter.FS       // target filesystem
	sd *slackdump.Session // slackdumper instance

	// time window
	opts Options
}

// Options allows to configure slack export options.
type Options struct {
	Oldest       time.Time
	Latest       time.Time
	IncludeFiles bool
}

// New creates a new Export instance, that will save export to the
// provided fs.
func New(sd *slackdump.Session, fs fsadapter.FS, cfg Options) *Export {
	return &Export{fs: fs, sd: sd, opts: cfg}
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

func (se *Export) users(ctx context.Context) (types.Users, error) {
	// fetch users and save them.
	users, err := se.sd.GetUsers(ctx)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (se *Export) messages(ctx context.Context, users types.Users) error {
	var chans []slack.Channel
	dl := downloader.New(se.sd.Client(), se.fs)
	if se.opts.IncludeFiles {
		// start the downloader
		dl.Start(ctx)
	}

	// we need the current user to be able to build an index of DMs.
	if err := se.sd.StreamChannels(ctx, slackdump.AllChanTypes, func(ch slack.Channel) error {
		if err := se.exportConversation(ctx, dl, users.IndexByID(), ch); err != nil {
			return err
		}

		chans = append(chans, ch)

		return nil

	}); err != nil {
		return fmt.Errorf("channels: error: %w", err)
	}

	idx, err := createIndex(chans, users, se.sd.CurrentUserID())
	if err != nil {
		return fmt.Errorf("failed to create an index: %w", err)
	}

	return idx.Marshal(se.fs)
}

// exportConversation exports one conversation.
func (se *Export) exportConversation(ctx context.Context, dl *downloader.Client, userIdx structures.UserIndex, ch slack.Channel) error {
	dlFn := se.downloadFn(dl, ch.Name)
	messages, err := se.sd.DumpMessagesRaw(ctx, ch.ID, se.opts.Oldest, se.opts.Latest, dlFn)
	if err != nil {
		return fmt.Errorf("failed to dump %q (%s): %w", ch.Name, ch.ID, err)
	}
	if len(messages.Messages) == 0 {
		// empty result set
		return nil
	}

	msgs, err := se.byDate(messages, userIdx)
	if err != nil {
		return fmt.Errorf("exportConversation: error: %w", err)
	}

	name, err := validName(ctx, ch, userIdx)
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
func (se *Export) downloadFn(dl *downloader.Client, channelName string) func(msg []types.Message, channelID string) (slackdump.ProcessResult, error) {
	const (
		entFiles  = "files"
		dirAttach = "attachments"
	)

	dir := filepath.Join(channelName, dirAttach)
	return func(msg []types.Message, channelID string) (slackdump.ProcessResult, error) {
		total := 0
		if err := files.Extract(msg, files.Root, func(file slack.File, addr files.Addr) error {
			filename, err := dl.DownloadFile(dir, file)
			if err != nil {
				return err
			}
			dlog.Debugf("submitted for download: %s", file.Name)
			total++
			return files.UpdateURLs(msg, addr, path.Join(dirAttach, path.Base(filename)))
		}); err != nil {
			if errors.Is(err, downloader.ErrNotStarted) {
				return slackdump.ProcessResult{Entity: entFiles, Count: 0}, nil
			}
			return slackdump.ProcessResult{}, err
		}

		return slackdump.ProcessResult{Entity: entFiles, Count: total}, nil
	}
}

// validName returns the channel or user name. Following the naming convention
// described by @niklasdahlheimer in this post (thanks to @Neznakomec for
// discovering it):
// https://github.com/RocketChat/Rocket.Chat/issues/13905#issuecomment-477500022
func validName(ctx context.Context, ch slack.Channel, uidx structures.UserIndex) (string, error) {
	if ch.IsIM {
		return ch.ID, nil
	} else {
		return ch.NameNormalized, nil
	}
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
