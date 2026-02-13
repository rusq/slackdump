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
		return elPre("slack-section-text", b.Text.Text), "", nil
	}
	if len(b.Fields) > 0 {
		var buf strings.Builder
		text := make([]string, len(b.Fields))
		for i, f := range b.Fields {
			text[i] = f.Text
		}
		buf.WriteString(elPre("slack-section-text", strings.Join(text, "\n")))
		return buf.String(), "", nil
	}
	return "", "", nil
}
