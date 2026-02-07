// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package renderer

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/osext"
	"github.com/rusq/slackdump/v3/internal/viewer/renderer/functions"
)

const debug = true

type Slack struct {
	tmpl    *template.Template
	uu      map[string]slack.User    // map of user id to user
	cc      map[string]slack.Channel // map of channel id to channel
	wspHost string                   // workspace URL to replace links to local
	host    string                   // host to replace links to local
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

func WithReplaceURL(wspURL, localHost string) SlackOption {
	return func(sm *Slack) {
		if localHost == "" {
			return
		}
		u, err := url.Parse(wspURL)
		if err != nil {
			slog.Warn("error parsing workspace URL", "error", err)
			return
		}
		sm.wspHost = u.Hostname()
		sm.host = localHost
	}
}

//go:embed templates/*.html
var templates embed.FS

func NewSlack(tmpl *template.Template, opts ...SlackOption) *Slack {
	s := &Slack{
		tmpl: template.Must(tmpl.New("blocks").Funcs(functions.FuncMap).ParseFS(templates, "templates/*.html")),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (*Slack) RenderText(ctx context.Context, s string) (v string) {
	return parseSlackMd(s)
}

func (s *Slack) Render(ctx context.Context, m *slack.Message) (v template.HTML) {
	var buf strings.Builder

	if len(m.Blocks.BlockSet) == 0 {
		fmt.Fprint(&buf, parseSlackMd(m.Text))
	} else {
		s.renderBlocks(ctx, &buf, m.Timestamp, m.Blocks.BlockSet)
	}
	s.renderFiles(ctx, &buf, m.Timestamp, m.Files)
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
		html, cl, err := fn(s, b)
		if err != nil {
			slog.ErrorContext(ctx, "error rendering block", "error", err, "block_type", b.BlockType(), attrMsgID)
			maybeprint(b)
			continue
		}
		buf.WriteString(html)
		buf.WriteString(cl)
	}
}

func (s *Slack) renderAttachments(ctx context.Context, buf *strings.Builder, msgTS string, attachments []slack.Attachment) {
	for _, a := range attachments {
		s.renderAttachment(ctx, buf, msgTS, a)
	}
}

func (s *Slack) renderAttachment(ctx context.Context, buf *strings.Builder, msgTS string, a slack.Attachment) {
	attrMsgID := slog.String("message_ts", msgTS)
	if err := s.tmpl.ExecuteTemplate(buf, "attachment.html", a); err != nil {
		slog.ErrorContext(ctx, "error rendering attachment", "error", err, attrMsgID)
	}
}

func (s *Slack) renderFiles(ctx context.Context, buf *strings.Builder, msgTS string, files []slack.File) {
	attrMsgID := slog.String("message_ts", msgTS)
	if files == nil {
		return
	}
	if err := s.tmpl.ExecuteTemplate(buf, "file.html", files); err != nil {
		slog.ErrorContext(ctx, "error rendering files", "error", err, attrMsgID)
	}
}

func maybeprint(v any) {
	if debug {
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		if err := enc.Encode(v); err != nil {
			log.Printf("error printing value: %s", err)
		}
		if err := os.Stderr.Sync(); err != nil {
			log.Printf("error flushing stderr: %s", err)
		}
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
	elBlockquote = element("blockquote", true)
	elDiv        = element("div", true)
	elFigure     = element("figure", true)
	elH3         = element("h3", true)
	elPre        = element("pre", true)
	elStrong     = element("strong", true)
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
