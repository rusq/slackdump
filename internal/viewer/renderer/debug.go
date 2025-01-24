package renderer

import (
	"context"
	"encoding/json"
	"html"
	"html/template"

	"github.com/rusq/slack"
)

type Debug struct{}

func (d *Debug) RenderText(ctx context.Context, s string) (v string) {
	return "<pre>" + html.EscapeString(s) + "</pre>"
}

func (d *Debug) Render(ctx context.Context, m *slack.Message) (v template.HTML) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err)
	}
	return template.HTML("<pre>" + html.EscapeString(m.Text) + "</pre><hr><code><pre>" + string(b) + "</pre></code>")
}
