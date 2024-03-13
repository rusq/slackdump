package renderer

import (
	"encoding/json"
	"html"
	"html/template"

	"github.com/rusq/slack"
)

type Debug struct{}

func (d *Debug) RenderText(s string) (v template.HTML) {
	return template.HTML("<pre>" + html.EscapeString(s) + "</pre>")
}

func (d *Debug) Render(m *slack.Message) (v template.HTML) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err)
	}
	return template.HTML("<pre>" + html.EscapeString(m.Text) + "</pre><hr><code><pre>" + string(b) + "</pre></code>")
}
