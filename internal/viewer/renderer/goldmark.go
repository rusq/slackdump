package renderer

import (
	"html/template"
	"log/slog"
	"strings"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	gparser "github.com/yuin/goldmark/parser"
	ghtml "github.com/yuin/goldmark/renderer/html"
)

type Goldmark struct {
	r goldmark.Markdown
}

func NewGoldmark() *Goldmark {
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
	return &Goldmark{r: md}
}

func (g *Goldmark) Render(s string) (v template.HTML) {
	var buf strings.Builder
	if err := g.r.Convert([]byte(s), &buf); err != nil {
		slog.Debug("error", "error", err)
		return template.HTML(s)
	}
	return template.HTML(buf.String())
}
