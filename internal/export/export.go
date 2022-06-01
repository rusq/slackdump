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
)

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

	idx, err := createIndex(chans, users, se.dumper.Me())
	if err != nil {
		return fmt.Errorf("failed to create an index: %w", err)
	}

	return idx.Marshal(se.fs)
}

// exportConversation exports one conversation.
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

// refRoot is the parent address for the topmost message chunk.
const refRoot = -1

// downloadFn returns the process function that should be passed to
// DumpMessagesRaw that will handle the download of the files.  If the
// downloader is not started, i.e. if file download is disabled, it will
// silently ignore the error and return nil.
func (se *Export) downloadFn(dl *downloader.Client, channelName string) func(msg []slackdump.Message, channelID string) (slackdump.ProcessResult, error) {
	const (
		entFiles  = "files"
		dirAttach = "attachments"
	)

	dir := filepath.Join(se.basedir(channelName), dirAttach)
	return func(msg []slackdump.Message, channelID string) (slackdump.ProcessResult, error) {
		total := 0
		if err := extractFiles(msg, refRoot, func(file slack.File, addr fileAddr) error {
			filename, err := dl.DownloadFile(dir, file)
			if err != nil {
				return err
			}
			dlog.Debugf("submitted for download: %s", file.Name)
			total++
			return updateFileURL(msg, addr, path.Join(dirAttach, path.Base(filename)))
		}); err != nil {
			if errors.Is(err, downloader.ErrNotStarted) {
				return slackdump.ProcessResult{Entity: entFiles, Count: 0}, nil
			}
			return slackdump.ProcessResult{}, err
		}

		return slackdump.ProcessResult{Entity: entFiles, Count: total}, nil
	}
}

// updateFileURL updates the URL link for the files in message chunk msgs.  Addr contains
// an address of the message to update the URL for, and filename is the path to the file
// on the local filesystem.  It will return an error if the address references out of range.
func updateFileURL(msgs []slackdump.Message, addr fileAddr, filename string) error {
	if addr.idxParMsg == refRoot {
		if addr.idxMsg < 0 || len(msgs) <= addr.idxMsg {
			return errors.New("invalid message reference")
		}
		if addr.idxFile < 0 || len(msgs[addr.idxMsg].Files) < addr.idxFile {
			return errors.New("invalid file reference")
		}
		msgs[addr.idxMsg].Files[addr.idxFile].URLPrivateDownload = filename
		msgs[addr.idxMsg].Files[addr.idxFile].URLPrivate = filename
	} else {
		return updateFileURL(
			msgs[addr.idxParMsg].ThreadReplies,
			fileAddr{idxMsg: addr.idxMsg, idxParMsg: refRoot, idxFile: addr.idxFile},
			filename,
		)
	}
	return nil
}

// fileAddr is the address of the file in the messages slice.
//   idxMsg    - index of the message or message reply in the provided slice
//   idxParMsg - index of the parent message. If it is not equal to refRoot, then
//               it is assumed that it is the reference to a message reply:
//               i.e.: msg[idxParMsg].ThreadReplies[idxMsg].
//   idxFile   - index of the file ine the message's file slice.
type fileAddr struct {
	idxMsg    int // index of the message in the messages slice
	idxParMsg int // index of the parent message, or refRoot if it's the address of the top level message
	idxFile   int // index of the file in the file slice.
}

// extractFiles scans the message slice msgs, and calls fn for each file it
// finds. fn is called with the copy of the file and that file's address in the
// provided message slice.  idxParentMsg is the index of the parent message (for
// message replies slice), or refRoot if it's the topmost messages slice (see
// invocation in downloadFn).
func extractFiles(msgs []slackdump.Message, idxParentMsg int, fn func(file slack.File, addr fileAddr) error) error {
	for mi := range msgs {
		if len(msgs[mi].Files) > 0 {
			for fileIdx, file := range msgs[mi].Files {
				if err := fn(file, fileAddr{idxMsg: mi, idxParMsg: idxParentMsg, idxFile: fileIdx}); err != nil {
					return err
				}
			}
		}
		if len(msgs[mi].ThreadReplies) > 0 {
			if err := extractFiles(msgs[mi].ThreadReplies, mi, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

// validName returns the channel or user name. Following the naming convention
// described by @niklasdahlheimer in this post (thanks to @Neznakomec for
// discovering it):
// https://github.com/RocketChat/Rocket.Chat/issues/13905#issuecomment-477500022
func validName(ctx context.Context, ch slack.Channel, uidx userIndex) (string, error) {
	if ch.IsIM {
		return ch.ID, nil
	} else {
		return ch.NameNormalized, nil
	}
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
