package renderer

import (
	"encoding/json"
	"errors"
	"html"
	"html/template"
	"log/slog"
	"os"
	"strings"

	"github.com/rusq/slack"
)

const debug = true

type Slack struct{}

func (*Slack) RenderText(s string) (v template.HTML) {
	// TODO parse legacy markdown
	return template.HTML("<pre>" + html.EscapeString(s) + "</pre>")
}

func (sm *Slack) Render(m *slack.Message) (v template.HTML) {
	if len(m.Blocks.BlockSet) == 0 {
		return sm.RenderText(m.Text)
	}

	attrMsgID := slog.String("message_ts", m.Timestamp)

	var buf strings.Builder
	for _, b := range m.Blocks.BlockSet {
		fn, ok := blockAction[b.BlockType()]
		if !ok {
			slog.Warn("unhandled block type", "block_type", b.BlockType(), attrMsgID)
			maybeprint(b)
			continue
		}
		s, err := fn(b)
		if err != nil {
			slog.Error("error rendering block", "error", err, "block_type", b.BlockType(), attrMsgID)
			maybeprint(b)
			continue
		}
		buf.WriteString(string(s))
	}
	return template.HTML(buf.String())
}

func maybeprint(b slack.Block) {
	if debug {
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		enc.Encode(b)
		os.Stderr.Sync()
	}
}

var blockAction = map[slack.MessageBlockType]func(slack.Block) (string, error){
	slack.MBTRichText: mbtRichText,
	slack.MBTImage:    mbtImage,
	slack.MBTContext:  mbtContext,
}

// ErrIncorrectBlockType is returned when the block type is not handled.
var ErrIncorrectBlockType = errors.New("incorrect block type")
