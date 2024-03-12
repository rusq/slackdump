package viewer

import (
	"html"
	"html/template"
	"log/slog"
	"strings"

	"github.com/rusq/slack"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	gparser "github.com/yuin/goldmark/parser"
	ghtml "github.com/yuin/goldmark/renderer/html"
)

type Renderer interface {
	RenderText(s string) (v template.HTML)
	Render(m *slack.Message) (v template.HTML)
}

type goldmrk struct {
	r goldmark.Markdown
}

func newGold() *goldmrk {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, emoji.Emoji, extension.DefinitionList),
		goldmark.WithParserOptions(
			gparser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			ghtml.WithHardWraps(),
			ghtml.WithXHTML(),
		),
	)
	return &goldmrk{r: md}
}

func (g *goldmrk) Render(s string) (v template.HTML) {
	var buf strings.Builder
	if err := g.r.Convert([]byte(s), &buf); err != nil {
		slog.Debug("error", "error", err)
		return template.HTML(s)
	}
	return template.HTML(buf.String())
}

type debugrender struct{}

func (d *debugrender) RenderText(s string) (v template.HTML) {
	return template.HTML("<pre>" + html.EscapeString(s) + "</pre>")
}

func (d *debugrender) Render(m *slack.Message) (v template.HTML) {
	return template.HTML("<pre>" + html.EscapeString(m.Text) + "</pre>")
}
