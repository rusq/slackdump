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

	"github.com/rusq/slack"
)

func (*Slack) mbtImage(ib slack.Block) (string, string, error) {
	b, ok := ib.(*slack.ImageBlock)
	if !ok {
		return "", "", NewErrIncorrectType(&slack.ImageBlock{}, ib)
	}
	return elFigure(
		blockTypeClass[slack.MBTImage],
		fmt.Sprintf(
			`<img src="%[1]s" alt="%[2]s"><figcaption class="slack-image-caption">%[2]s</figcaption>`,
			b.ImageURL, b.AltText,
		),
	), "", nil
}
