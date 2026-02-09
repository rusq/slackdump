// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
	return elDiv("slack-actions", buf.String()), "", nil
}
