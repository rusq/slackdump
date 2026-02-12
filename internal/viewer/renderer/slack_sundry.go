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
