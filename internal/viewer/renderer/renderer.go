package renderer

import (
	"html/template"

	"github.com/rusq/slack"
)

type Renderer interface {
	RenderText(s string) (v template.HTML)
	Render(m *slack.Message) (v template.HTML)
}
