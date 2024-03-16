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
	tmpl *template.Template
	uu   map[string]slack.User    // map of user id to user
	cc   map[string]slack.Channel // map of channel id to channel
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

func NewSlack(tmpl *template.Template, opts ...SlackOption) *Slack {
	s := &Slack{
		tmpl: tmpl,
	}
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

	var buf strings.Builder
	s.renderBlocks(ctx, &buf, m.Timestamp, m.Blocks.BlockSet)
	s.renderAttachments(ctx, &buf, m.Timestamp, m.Attachments)

	return template.HTML(buf.String())
}

// renderBlocks renders the blocks to the buffer.  msgTS is used to identify
// the message which failed to render in the logs.
func (s *Slack) renderBlocks(ctx context.Context, buf *strings.Builder, msgTS string, blocks []slack.Block) {
	attrMsgID := slog.String("message_ts", msgTS)

	for _, b := range blocks {
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
}

func (s *Slack) renderAttachments(ctx context.Context, buf *strings.Builder, msgTS string, attachments []slack.Attachment) {
	attrMsgID := slog.String("message_ts", msgTS)
	for _, a := range attachments {
		if err := s.tmpl.ExecuteTemplate(buf, "attachment.html", a); err != nil {
			slog.ErrorContext(ctx, "error rendering attachment", "error", err, attrMsgID)
		}
	}
}

func maybeprint(v any) {
	if debug {
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		enc.Encode(v)
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
var (
	div        = element("div", true)
	figure     = element("figure", true)
	blockquote = element("blockquote", true)
)

func element(el string, close bool) func(class string, s string) string {
	return func(class, s string) string {
		var buf strings.Builder
		fmt.Fprintf(&buf, `<%s class="%s">%s`, el, class, s)
		if close {
			fmt.Fprintf(&buf, `</%s>`, el)
		}
		return buf.String()
	}
}
