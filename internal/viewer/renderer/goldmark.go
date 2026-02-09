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
