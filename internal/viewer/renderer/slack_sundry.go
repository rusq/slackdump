package renderer

import "github.com/rusq/slack"

/*
	{
	  "type": "call",
	  "block_id": "35d6f"
	},
*/
func (*Slack) mbtCall(slack.Block) (string, string, error) {
	return div("slack-call", "(Call)"), "", nil
}

func (*Slack) mbtDivider(slack.Block) (string, string, error) {
	return div("slack-divider", "<hr/>"), "", nil
}
