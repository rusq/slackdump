package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"runtime/trace"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/rusq/slackdump/v2/types"
)

// Export is the instance of Slack Exporter.
type Export struct {
	fs fsadapter.FS       // target filesystem
	sd *slackdump.Session // Session instance
	lg logger.Interface

	// time window
	opts Options
}

// New creates a new Export instance, that will save export to the
// provided fs.
func New(sd *slackdump.Session, fs fsadapter.FS, cfg Options) *Export {
	se := &Export{fs: fs, sd: sd, lg: cfg.Logger, opts: cfg}
	if se.lg == nil {
		se.lg = logger.Default
	}
	network.Logger = se.l()
	return se
}

// Run runs the export.
func (se *Export) Run(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "export.Run")
	defer task.End()

	// export users to users.json
	users, err := se.sd.GetUsers(ctx)
	if err != nil {
		trace.Logf(ctx, "error", "GetUsers: %s", err)
		return err
	}

	// export channels to channels.json
	if err := se.messages(ctx, users); err != nil {
		trace.Logf(ctx, "error", "messages: %s", err)
		return err
	}
	return nil
}

func (se *Export) messages(ctx context.Context, users types.Users) error {
	ctx, task := trace.NewTask(ctx, "export.messages")
	defer task.End()

	dl := downloader.New(se.sd.Client(), se.fs, downloader.Logger(se.l()))
	if se.opts.IncludeFiles {
		// start the downloader
		dl.Start(ctx)
		defer func() {
			trace.Log(ctx, "info", "waiting for downloads to finish")
			dl.Stop()
			trace.Log(ctx, "info", "downloader stopped")
		}()
	}

	var chans []slack.Channel

	chans, err := se.exportChannels(ctx, dl, users, se.opts.List)
	if err != nil {
		return fmt.Errorf("export error: %w", err)
	}

	idx, err := createIndex(chans, users, se.sd.CurrentUserID())
	if err != nil {
		return fmt.Errorf("failed to create an index: %w", err)
	}

	if err := idx.Marshal(se.fs); err != nil {
		return err
	}

	return nil
}

func (se *Export) exportChannels(ctx context.Context, dl *downloader.Client, users types.Users, el *structures.EntityList) ([]slack.Channel, error) {
	if se.opts.List.HasIncludes() {
		// if there an Include list, we don't need to retrieve all channels,
		// only the ones that are specified.
		return se.inclusiveExport(ctx, dl, users, se.opts.List)
	} else {
		return se.exclusiveExport(ctx, dl, users, se.opts.List)
	}
}

// exclusiveExport exports all channels, excluding ones that are defined in
// EntityList.  If EntityList has Include channels, they are ignored.
func (se *Export) exclusiveExport(ctx context.Context, dl *downloader.Client, users types.Users, el *structures.EntityList) ([]slack.Channel, error) {
	ctx, task := trace.NewTask(ctx, "export.exclusive")
	defer task.End()

	chans := make([]slack.Channel, 0)

	listIdx := el.Index()
	uidx := users.IndexByID()
	// we need the current user to be able to build an index of DMs.
	if err := se.sd.StreamChannels(ctx, slackdump.AllChanTypes, func(ch slack.Channel) error {
		if include, ok := listIdx[ch.ID]; ok && !include {
			trace.Logf(ctx, "info", "skipping %s", ch.ID)
			se.lg.Printf("skipping: %s", ch.ID)
			return nil
		}
		if err := se.exportConversation(ctx, dl, uidx, ch); err != nil {
			return err
		}

		chans = append(chans, ch)
		return nil

	}); err != nil {
		return nil, fmt.Errorf("channels: error: %w", err)
	}
	se.l().Printf("  out of which exported:  %d", len(chans))
	return chans, nil
}

// inclusiveExport exports only channels that are defined in the
// EntryList.Include.
func (se *Export) inclusiveExport(ctx context.Context, dl *downloader.Client, users types.Users, list *structures.EntityList) ([]slack.Channel, error) {
	ctx, task := trace.NewTask(ctx, "export.inclusive")
	defer task.End()

	if !list.HasIncludes() {
		return nil, errors.New("empty input")
	}

	// preallocate, some channels might be excluded, so this is optimistic
	// allocation
	chans := make([]slack.Channel, 0, len(list.Include))

	elIdx := list.Index()

	// we need the current user to be able to build an index of DMs.
	for _, entry := range list.Include {
		if include, ok := elIdx[entry]; ok && !include {
			trace.Logf(ctx, "info", "skipping %s", entry)
			se.lg.Printf("skipping: %s", entry)
			continue
		}
		sl, err := structures.ParseLink(entry)
		if err != nil {
			return nil, err
		}
		ch, err := se.sd.Client().GetConversationInfoContext(ctx, sl.Channel, true)
		if err != nil {
			return nil, fmt.Errorf("error getting info for %s: %w", sl, err)
		}

		if err := se.exportConversation(ctx, dl, users.IndexByID(), *ch); err != nil {
			return nil, err
		}

		chans = append(chans, *ch)
	}

	return chans, nil
}

// exportConversation exports one conversation.
func (se *Export) exportConversation(ctx context.Context, dl *downloader.Client, userIdx structures.UserIndex, ch slack.Channel) error {
	ctx, task := trace.NewTask(ctx, "export.conversation")
	defer task.End()

	dlFn := se.downloadFn(dl, ch.Name)
	messages, err := se.sd.DumpRaw(ctx, ch.ID, se.opts.Oldest, se.opts.Latest, dlFn)
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

	name, err := validName(ch, userIdx)
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
			se.l().Debugf("submitted for download: %s", file.Name)
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
func validName(ch slack.Channel, uidx structures.UserIndex) (string, error) {
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

func (se *Export) l() logger.Interface {
	if se.lg == nil {
		se.lg = logger.Default
	}
	return se.lg
}
