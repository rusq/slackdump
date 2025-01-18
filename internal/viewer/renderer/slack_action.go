package renderer

import (
	"fmt"
	"strings"

	"github.com/rusq/slack"
)

/*
{
  "type": "actions",
  "block_id": "{\"task_id\":\"1209021\",\"id\":\"........\"}",
  "elements": [
    {
      "type": "button",
      "text": {
        "type": "plain_text",
        "text": "View",
        "emoji": true
      },
      "action_id": "jira_view_modal",
      "value": "jira_view_modal"
    },
*/

func (*Slack) mbtAction(ib slack.Block) (string, string, error) {
	b, ok := ib.(*slack.ActionBlock)
	if !ok {
		return "", "", NewErrIncorrectType(&slack.ActionBlock{}, ib)
	}
	var buf strings.Builder
	for _, e := range b.Elements.ElementSet {
		switch e := e.(type) {
		case *slack.ButtonBlockElement:
			fmt.Fprintf(&buf, `<BUTTON alt="%s">%s</BUTTON>`, e.ActionID, e.Text.Text)
		default:
			fmt.Fprintf(&buf, "[ELEMENT: %T]", e)
		}
	}
	return div("slack-actions", buf.String()), "", nil
}
