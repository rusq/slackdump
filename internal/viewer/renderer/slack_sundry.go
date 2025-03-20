package renderer

import "github.com/rusq/slack"

/*
	{
	  "type": "call",
	  "block_id": "35d6f"
	},
*/
func (*Slack) mbtCall(slack.Block) (string, string, error) {
	return elDiv("slack-call", "(Call)"), "", nil
}

func (*Slack) mbtDivider(slack.Block) (string, string, error) {
	return elDiv("slack-divider", "<hr/>"), "", nil
}

/*
	{
	  "type": "header",
	  "text": {
	    "type": "plain_text",
	    "text": "Very long thread (1000 messages)",
	    "emoji": true
	  },
	  "block_id": "D17qa"
	}
*/
func (*Slack) mbtHeader(ib slack.Block) (string, string, error) {
	b, ok := ib.(*slack.HeaderBlock)
	if !ok {
		return "", "", NewErrIncorrectType(&slack.HeaderBlock{}, ib)
	}
	switch b.Text.Type {
	case slack.PlainTextType:
		return elH3("slack-header", b.Text.Text), "", nil
	default:
		return "", "", NewErrIncorrectType(slack.PlainTextType, b.Text.Type)
	}
}
