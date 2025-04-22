package transform

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"sort"
	"strings"

	"github.com/rusq/slackdump/v3/source"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/nametmpl"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/types"
)

type DumpOption func(*DumpConverter)

type Templater interface {
	Execute(c *types.Conversation) string
}

func DumpWithTemplate(tmpl Templater) DumpOption {
	return func(s *DumpConverter) {
		s.tmpl = tmpl
	}
}

// DumpWithPipeline adds a pipeline function to the transformer, that will
// be called for each message slice, before it is written to the filesystem.
func DumpWithPipeline(f ...func(channelID string, threadTS string, mm []slack.Message) error) DumpOption {
	return func(s *DumpConverter) {
		s.pipeline = append(s.pipeline, f...)
	}
}

func DumpWithLogger(log *slog.Logger) DumpOption {
	return func(s *DumpConverter) {
		s.lg = log
	}
}

// NewDump creates a new standard dump converter.
func NewDump(fsa fsadapter.FS, src source.Sourcer, opts ...DumpOption) (*DumpConverter, error) {
	std := &DumpConverter{
		src:  src,
		fsa:  fsa,
		lg:   slog.Default(),
		tmpl: nametmpl.NewDefault(),
	}
	for _, opt := range opts {
		opt(std)
	}
	return std, nil
}

// pipelineFunc is a function that performs caller-defined transformations of
// a slice of slack messages.  The type alias is defined for brevity.
type (
	pipelineFunc = func(channelID string, threadTS string, mm []slack.Message) error
	pipeline     []pipelineFunc
)

// apply applies the pipeline functions in order.
func (p pipeline) apply(channelID, threadTS string, mm []slack.Message) error {
	for i, f := range p {
		if err := f(channelID, threadTS, mm); err != nil {
			return fmt.Errorf("pipeline seq=%d, error: %w", i, err)
		}
	}
	return nil
}

// DumpConverter is a converter of chunk files into the Slackdump format.
type DumpConverter struct {
	src      source.Sourcer // source of the data
	fsa      fsadapter.FS   // output file system adapter
	tmpl     Templater      // file name template
	lg       *slog.Logger   // logger
	pipeline []pipelineFunc // pipeline filter functions
}

// Convert converts the chunk file to Slackdump json format.
func (s *DumpConverter) Convert(ctx context.Context, channelID, threadID string) error {
	ci, err := s.src.ChannelInfo(ctx, channelID)
	if err != nil {
		return err
	}
	slog.Debug("DumpConverter.Convert", "channel", channelID, "thread", threadID)

	var msgs []types.Message
	// threadTS is only populated on the thread only files.  It is safe to
	// rely on it being non-empty to determine if we need a thread or a
	// conversation.
	if threadID == "" {
		msgs, err = stdConversation(ctx, s.src, ci, s.pipeline)
	} else {
		msgs, err = stdThread(ctx, s.src, ci, threadID, s.pipeline)
	}
	if err != nil {
		return err
	}
	conv := &types.Conversation{
		ID:       ci.ID,
		Name:     ci.Name,
		ThreadTS: threadID,
		Messages: msgs,
	}

	f, err := s.fsa.Create(s.tmpl.Execute(conv))
	if err != nil {
		name := s.tmpl.Execute(conv)
		return fmt.Errorf("fsadapter: unable to create file %s: %w", name, err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(conv)
}

func collect[T any](it iter.Seq2[T, error], sz int) ([]T, error) {
	vs := make([]T, 0, sz)
	for c, err := range it {
		if err != nil {
			return nil, err
		}
		vs = append(vs, c)
	}
	return vs, nil
}

const (
	msgPrealloc  = 10000
	tmsgPrealloc = 5
)

// stdConversation is the function that does the transformation of the whole
// channel with threads.
func stdConversation(ctx context.Context, cf source.Sourcer, ci *slack.Channel, pipeline pipeline) ([]types.Message, error) {
	it, err := cf.AllMessages(ctx, ci.ID)
	if err != nil {
		return nil, err
	}
	mm, err := collect(it, msgPrealloc)
	if err != nil {
		return nil, err
	}
	sort.Sort(structures.Messages(mm))
	if err := pipeline.apply(ci.ID, "", mm); err != nil {
		return nil, fmt.Errorf("conversation: %w", err)
	}
	msgs := make([]types.Message, 0, len(mm))
	for i := range mm {
		if strings.EqualFold(mm[i].SubType, structures.SubTypeThreadBroadcast) {
			// this we don't eat.  Skip thread broadcasts.
			continue
		}
		var sdm types.Message // slackdump message
		sdm.Message = mm[i]
		if mm[i].ThreadTimestamp != "" && structures.IsThreadStart(&mm[i]) && mm[i].LatestReply != structures.LatestReplyNoReplies {
			it, err := cf.AllThreadMessages(ctx, ci.ID, mm[i].ThreadTimestamp)
			if err != nil {
				return nil, err
			}
			// process thread only for parent messages
			// if there's a thread timestamp, we need to find and add it.
			thread, err := collect(it, tmsgPrealloc)
			if err != nil {
				return nil, err
			}
			sdm.ThreadReplies = types.ConvertMsgs(thread)
		}
		msgs = append(msgs, sdm)
	}
	return msgs, nil
}

// stdThread is the function that performs transformation of a single thread.
func stdThread(ctx context.Context, src source.Sourcer, ci *slack.Channel, threadTS string, pipeline pipeline) ([]types.Message, error) {
	// this is a thread.
	it, err := src.AllThreadMessages(ctx, ci.ID, threadTS)
	if err != nil {
		return nil, err
	}
	mm, err := collect(it, tmsgPrealloc)
	if err != nil {
		return nil, err
	}

	if err := pipeline.apply(ci.ID, threadTS, mm); err != nil {
		return nil, fmt.Errorf("conversation: %w", err)
	}

	// sort messages by timestamp
	sort.Sort(structures.Messages(mm))

	return types.ConvertMsgs(mm), nil
}

// Users writes the users to the filesystem.
func (dw *DumpConverter) Users(_ context.Context, uu []slack.User) error {
	return marshalFormatted(dw.fsa, "users.json", uu)
}

// Channels writes the channels to the filesystem.
func (dw *DumpConverter) Channels(_ context.Context, cc []slack.Channel) error {
	return marshalFormatted(dw.fsa, "channels.json", cc)
}

// WorkspaceInfo writes the workspace info to the filesystem.
func (dw *DumpConverter) WorkspaceInfo(_ context.Context, wi *slack.AuthTestResponse) error {
	return marshalFormatted(dw.fsa, "workspace.json", wi)
}

// marshalFormatted writes the data to the filesystem in a formatted way.
func marshalFormatted(fsa fsadapter.FS, filename string, a any) error {
	f, err := fsa.Create(filename)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", " ")
	if err := enc.Encode(a); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}
