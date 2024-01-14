package transform

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rusq/fsadapter"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/nametmpl"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/rusq/slackdump/v3/types"
)

type StdOption func(*StdConverter)

type Templater interface {
	Execute(c *types.Conversation) string
}

func StdWithTemplate(tmpl Templater) StdOption {
	return func(s *StdConverter) {
		s.tmpl = tmpl
	}
}

// StdWithPipeline adds a pipeline function to the transformer, that will
// be called for each message slice, before it is written to the filesystem.
func StdWithPipeline(f ...func(channelID string, threadTS string, mm []slack.Message) error) StdOption {
	return func(s *StdConverter) {
		s.pipeline = append(s.pipeline, f...)
	}
}

func StdWithLogger(log logger.Interface) StdOption {
	return func(s *StdConverter) {
		s.lg = log
	}
}

// NewStandard creates a new standard dump transformer.
func NewStandard(fsa fsadapter.FS, dir string, opts ...StdOption) (*StdConverter, error) {
	cd, err := chunk.OpenDir(dir)
	if err != nil {
		return nil, err
	}
	defer cd.Close()
	std := &StdConverter{
		cd:   cd,
		fsa:  fsa,
		lg:   logger.Default,
		tmpl: nametmpl.NewDefault(),
	}
	for _, opt := range opts {
		opt(std)
	}
	return std, nil
}

// pipelineFunc is a function that transforms a slice of slack messages.
// defining a type alias for brevity.
type pipelineFunc = func(channelID string, threadTS string, mm []slack.Message) error
type pipeline []pipelineFunc

func (p pipeline) apply(channelID, threadTS string, mm []slack.Message) error {
	for i, f := range p {
		if err := f(channelID, threadTS, mm); err != nil {
			return fmt.Errorf("pipeline seq=%d, error: %w", i, err)
		}
	}
	return nil
}

type StdConverter struct {
	cd       *chunk.Directory
	fsa      fsadapter.FS
	tmpl     Templater
	lg       logger.Interface
	pipeline []pipelineFunc
}

func (s *StdConverter) Convert(ctx context.Context, id chunk.FileID) error {
	cf, err := s.cd.Open(id)
	if err != nil {
		return err
	}
	defer cf.Close()

	// threadTS is only populated on the thread only files.  It is safe to
	// rely on it being non-empty to determine if we need a thread or a
	// conversation.
	channelID, threadID := id.Split()
	ci, err := cf.ChannelInfo(channelID)
	if err != nil {
		return err
	}

	var msgs []types.Message
	if threadID == "" {
		msgs, err = stdConversation(cf, ci, s.pipeline)
	} else {
		msgs, err = stdThread(cf, ci, threadID, s.pipeline)
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
		return fmt.Errorf("fsadapter: unable to create file %s: %w", id+".json", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(conv)
}

// stdConversation is the function that does the transformation of the whole
// channel with threads.
func stdConversation(cf *chunk.File, ci *slack.Channel, pipeline pipeline) ([]types.Message, error) {
	mm, err := cf.AllMessages(ci.ID)
	if err != nil {
		return nil, err
	}
	if err := pipeline.apply(ci.ID, "", mm); err != nil {
		return nil, fmt.Errorf("conversation: %w", err)
	}
	msgs := make([]types.Message, 0, len(mm))
	for i := range mm {
		if mm[i].SubType == "thread_broadcast" {
			// this we don't eat.  Skip thread broadcasts.
			continue
		}
		var sdm types.Message // slackdump message
		sdm.Message = mm[i]
		if mm[i].ThreadTimestamp != "" && mm[i].ThreadTimestamp == mm[i].Timestamp && mm[i].LatestReply != structures.NoRepliesLatestReply { // process thread only for parent messages
			// if there's a thread timestamp, we need to find and add it.
			thread, err := cf.AllThreadMessages(ci.ID, mm[i].ThreadTimestamp)
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
func stdThread(cf *chunk.File, ci *slack.Channel, threadTS string, pipeline pipeline) ([]types.Message, error) {
	// this is a thread.
	mm, err := cf.AllThreadMessages(ci.ID, threadTS)
	if err != nil {
		return nil, err
	}
	// get parent message
	parent, err := cf.ThreadParent(ci.ID, threadTS) // TODO: implement
	if err != nil {
		return nil, err
	}
	mm = append([]slack.Message{*parent}, mm...)

	if err := pipeline.apply(ci.ID, threadTS, mm); err != nil {
		return nil, fmt.Errorf("conversation: %w", err)
	}

	return types.ConvertMsgs(mm), nil
}
