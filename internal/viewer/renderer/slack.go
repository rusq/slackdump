package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"strings"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/osext"
)

const debug = true

type Slack struct {
	uu map[string]slack.User    // map of user id to user
	cc map[string]slack.Channel // map of channel id to channel
}

type SlackOption func(*Slack)

func WithUsers(uu map[string]slack.User) SlackOption {
	return func(sm *Slack) {
		sm.uu = uu
	}
}

func WithChannels(cc map[string]slack.Channel) SlackOption {
	return func(sm *Slack) {
		sm.cc = cc
	}
}

func NewSlack(opts ...SlackOption) *Slack {
	s := &Slack{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (*Slack) RenderText(ctx context.Context, s string) (v template.HTML) {
	return template.HTML(parseSlackMd(s))
}

func (s *Slack) Render(ctx context.Context, m *slack.Message) (v template.HTML) {
	if len(m.Blocks.BlockSet) == 0 {
		return s.RenderText(ctx, m.Text)
	}

	attrMsgID := slog.String("message_ts", m.Timestamp)

	var buf strings.Builder
	for _, b := range m.Blocks.BlockSet {
		fn, ok := blockTypeHandlers[b.BlockType()]
		if !ok {
			slog.WarnContext(ctx, "unhandled block type", "block_type", b.BlockType(), attrMsgID)
			maybeprint(b)
			continue
		}
		html, err := fn(s, b)
		if err != nil {
			slog.ErrorContext(ctx, "error rendering block", "error", err, "block_type", b.BlockType(), attrMsgID)
			maybeprint(b)
			continue
		}
		buf.WriteString(html)
	}
	return template.HTML(buf.String())
}

func maybeprint(b slack.Block) {
	if debug {
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		enc.Encode(b)
		os.Stderr.Sync()
	}
}

const stackframe = 1

type ErrIncorrectBlockType struct {
	Caller string
	Want   any
	Got    any
}

func (e *ErrIncorrectBlockType) Error() string {
	return fmt.Sprintf("incorrect block type for block %s: want %T, got %T", e.Caller, e.Want, e.Got)
}

func NewErrIncorrectType(want, got any) error {
	return &ErrIncorrectBlockType{
		Caller: osext.Caller(stackframe),
		Want:   want,
		Got:    got,
	}
}

type ErrMissingHandler struct {
	Caller string
	Type   any
}

func (e *ErrMissingHandler) Error() string {
	return fmt.Sprintf("missing handler for type %v called in %s", e.Type, e.Caller)
}

func NewErrMissingHandler(t any) error {
	return &ErrMissingHandler{
		Caller: osext.Caller(stackframe),
		Type:   t,
	}
}

// classes
func div(class string, s string) string {
	return fmt.Sprintf(`<div class=\"%s\">%s</div>`, class, s)
}
