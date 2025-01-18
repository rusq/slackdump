package renderer

import (
	"strings"

	"github.com/rusq/slack"
)

/*
   {
     "type": "section",
     "text": {
       "type": "mrkdwn",
       "text": "Meeting passcode: yyyyy"
     },
     "block_id": "swSOO"
   }
*/

func (*Slack) mbtSection(ib slack.Block) (string, string, error) {
	b, ok := ib.(*slack.SectionBlock)
	if !ok {
		return "", "", NewErrIncorrectType(&slack.SectionBlock{}, ib)
	}
	if b.Text != nil {
		return pre("slack-section-text", b.Text.Text), "", nil
	}
	if len(b.Fields) > 0 {
		var buf strings.Builder
		text := make([]string, len(b.Fields))
		for i, f := range b.Fields {
			text[i] = f.Text
		}
		buf.WriteString(pre("slack-section-text", strings.Join(text, "\n")))
		return buf.String(), "", nil
	}
	return "", "", nil
}
