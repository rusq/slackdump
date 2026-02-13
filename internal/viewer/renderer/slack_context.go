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

func (s *Slack) mbtContext(ib slack.Block) (string, string, error) {
	b, ok := ib.(*slack.ContextBlock)
	if !ok {
		return "", "", NewErrIncorrectType(&slack.ContextBlock{}, ib)
	}
	var buf, cbuf strings.Builder
	for _, el := range b.ContextElements.Elements {
		fn, ok := contextElementHandlers[el.MixedElementType()]
		if !ok {
			return "", "", NewErrMissingHandler(el.MixedElementType())
		}
		s, cl, err := fn(s, el)
		if err != nil {
			return "", "", err
		}
		buf.WriteString(s)
		cbuf.WriteString(cl)
	}

	return buf.String(), cbuf.String(), nil
}

var contextElementHandlers = map[slack.MixedElementType]func(*Slack, slack.MixedElement) (string, string, error){
	slack.MixedElementImage: (*Slack).metImage,
	slack.MixedElementText:  (*Slack).metText,
}

func (*Slack) metImage(ie slack.MixedElement) (string, string, error) {
	e, ok := ie.(*slack.ImageBlockElement)
	if !ok {
		return "", "", NewErrIncorrectType(&slack.ImageBlockElement{}, ie)
	}
	uri := ""
	if e.ImageURL != nil {
		uri = *e.ImageURL
	}
	return fmt.Sprintf(`<img src="%s" alt="%s">`, uri, e.AltText), "", nil
}

func (*Slack) metText(ie slack.MixedElement) (string, string, error) {
	e, ok := ie.(*slack.TextBlockObject)
	if !ok {
		return "", "", NewErrIncorrectType(&slack.TextBlockObject{}, ie)
	}
	return e.Text, "", nil
}
