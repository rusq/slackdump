package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime/trace"

	"github.com/rusq/slack"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/internal/structures/files/dl"
	"github.com/rusq/slackdump/v3/types"
)

// Export is the instance of Slack Exporter.
type Export struct {
	fs fsadapter.FS // target filesystem
	sd dumper       // Session instance
	lg *slog.Logger
	dl dl.Exporter

	// options
	opts Config
}

// New creates a new Export instance, that will save export to the
// provided fs.
func New(sd *slackdump.Session, fs fsadapter.FS, cfg Config) *Export {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	se := &Export{
		fs:   fs,
		sd:   sd,
		lg:   cfg.Logger,
		opts: cfg,
		dl:   newV2FileExporter(cfg.Type, fs, sd.Client(), cfg.Logger, cfg.ExportToken),
	}
	return se
}

// Run runs the export.
func (se *Export) Run(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "export.Run")
	defer task.End()

	// export users to users.json
	users, err := se.sd.GetUsers(ctx)
	if err != nil {
		se.td(ctx, "error", "GetUsers: %s", err)
		return err
	}

	// export channels to channels.json
	if err := se.messages(ctx, users); err != nil {
		se.td(ctx, "error", "messages: %s", err)
		return err
	}
	return nil
}

func (se *Export) messages(ctx context.Context, users types.Users) error {
	ctx, task := trace.NewTask(ctx, "export.messages")
	defer task.End()

	if se.opts.IsFilesEnabled() {
		// start the dl
		se.dl.Start(ctx)
		defer func() {
			se.td(ctx, "info", "waiting for downloads to finish")
			se.lg.DebugContext(ctx, "waiting for downloads to finish")
			se.dl.Stop()
			se.td(ctx, "info", "dl stopped")
			se.lg.DebugContext(ctx, "downloader stopped")
		}()
	}

	var chans []slack.Channel

	chans, err := se.exportChannels(ctx, users.IndexByID())
	if err != nil {
		return fmt.Errorf("export error: %w", err)
	}

	idx, err := structures.MakeExportIndex(chans, users, se.sd.CurrentUserID())
	if err != nil {
		return fmt.Errorf("failed to create an index: %w", err)
	}

	if err := idx.Marshal(se.fs); err != nil {
		return err
	}

	return nil
}

func (se *Export) exportChannels(ctx context.Context, uidx structures.UserIndex) ([]slack.Channel, error) {
	if se.opts.List != nil && se.opts.List.HasIncludes() {
		// if there's an "Include" list, we don't need to retrieve all channels,
		// only the ones that are specified.
		return se.inclusiveExport(ctx, uidx, se.opts.List)
	} else {
		return se.exclusiveExport(ctx, uidx, se.opts.List)
	}
}

// exclusiveExport exports all channels, excluding ones that are defined in
// EntityList.  If EntityList has Include channels, they are ignored.
func (se *Export) exclusiveExport(ctx context.Context, uidx structures.UserIndex, el *structures.EntityList) ([]slack.Channel, error) {
	ctx, task := trace.NewTask(ctx, "export.exclusive")
	defer task.End()

	chans := make([]slack.Channel, 0)

	listIdx := el.Index()
	// we need the current user to be able to build an index of DMs.
	if err := se.sd.StreamChannels(ctx, slackdump.AllChanTypes, func(ch slack.Channel) error {
		if listIdx.IsExcluded(ch.ID) {
			trace.Logf(ctx, "info", "skipping %s", ch.ID)
			se.lg.InfoContext(ctx, "skipping (excluded)", "channel_id", ch.ID)
			return nil
		}
		if err := se.exportConversation(ctx, uidx, ch); err != nil {
			return err
		}

		chans = append(chans, ch)
		return nil

	}); err != nil {
		return nil, fmt.Errorf("channels: error: %w", err)
	}
	se.lg.InfoContext(ctx, "  out of which exported:  ", "n", len(chans))
	return chans, nil
}

// inclusiveExport exports only channels that are defined in the
// EntryList.Include.
func (se *Export) inclusiveExport(ctx context.Context, uidx structures.UserIndex, list *structures.EntityList) ([]slack.Channel, error) {
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
		if elIdx.IsExcluded(entry) {
			se.td(ctx, "info", "skipping %s", entry)
			se.lg.InfoContext(ctx, "skipping (excluded)", "entry", entry)
			continue
		}
		sl, err := structures.ParseLink(entry)
		if err != nil {
			return nil, err
		}
		ch, err := se.sd.Client().GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{ChannelID: sl.Channel, IncludeLocale: true, IncludeNumMembers: true})
		if err != nil {
			return nil, fmt.Errorf("error getting info for %s: %w", sl, err)
		}

		if err := se.exportConversation(ctx, uidx, *ch); err != nil {
			return nil, err
		}

		chans = append(chans, *ch)
	}

	return chans, nil
}

// exportConversation exports one conversation.
func (se *Export) exportConversation(ctx context.Context, userIdx structures.UserIndex, ch slack.Channel) error {
	ctx, task := trace.NewTask(ctx, "export.conversation")
	defer task.End()

	messages, err := se.sd.DumpRaw(ctx, ch.ID, se.opts.Oldest, se.opts.Latest, se.dl.ProcessFunc(validName(ch)))
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

	name := validName(ch)

	if err := se.saveChannel(name, msgs); err != nil {
		return err
	}

	return nil
}

// validName returns the channel or user name. Following the naming convention
// described by @niklasdahlheimer in this post (thanks to @Neznakomec for
// discovering it):
// https://github.com/RocketChat/Rocket.Chat/issues/13905#issuecomment-477500022
func validName(ch slack.Channel) string {
	if ch.IsIM {
		return ch.ID
	}
	return ch.Name
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

// td outputs the message to trace and logs a debug message.
func (se *Export) td(ctx context.Context, category string, fmt string, a ...any) {
	trace.Logf(ctx, category, fmt, a...)
}
