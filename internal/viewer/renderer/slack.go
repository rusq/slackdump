package renderer

import (
	"errors"
	"html"
	"html/template"
	"log/slog"
	"strings"

	"github.com/rusq/slack"
)

type Slack struct{}

func (*Slack) RenderText(s string) (v template.HTML) {
	// TODO parse legacy markdown
	return template.HTML("<pre>" + html.EscapeString(s) + "</pre>")
}

func (sm *Slack) Render(m *slack.Message) (v template.HTML) {
	if len(m.Blocks.BlockSet) == 0 {
		return sm.RenderText(m.Text)
	}
	var buf strings.Builder
	for _, b := range m.Blocks.BlockSet {
		fn, ok := blockAction[b.BlockType()]
		if !ok {
			slog.Warn("unhandled block type", "block_type", b.BlockType())
			continue
		}
		s, err := fn(b)
		if err != nil {
			slog.Error("error rendering block", "error", err, "block_type", b.BlockType())
			continue
		}
		buf.WriteString(string(s))
	}
	return template.HTML(buf.String())
}

var blockAction = map[slack.MessageBlockType]func(slack.Block) (string, error){
	slack.MBTRichText: mbtRichText,
}

// ErrIncorrectBlockType is returned when the block type is not handled.
var ErrIncorrectBlockType = errors.New("incorrect block type")
