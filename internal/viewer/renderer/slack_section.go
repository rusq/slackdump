package renderer

import (
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

func (*Slack) mbtSection(ib slack.Block) (string, error) {
	b, ok := ib.(*slack.SectionBlock)
	if !ok {
		return "", NewErrIncorrectType(&slack.SectionBlock{}, ib)
	}
	return pre("slack-section-text", b.Text.Text), nil
}
