package renderer

import (
	"context"
	"html/template"

	"github.com/rusq/slack"
)

type Renderer interface {
	RenderText(ctx context.Context, s string) (v string)
	Render(ctx context.Context, m *slack.Message) (v template.HTML)
}
