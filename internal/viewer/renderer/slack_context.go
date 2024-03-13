package renderer

import (
	"fmt"
	"strings"

	"github.com/rusq/slack"
)

func mbtContext(ib slack.Block) (string, error) {
	b, ok := ib.(*slack.ContextBlock)
	if !ok {
		return "", ErrIncorrectBlockType
	}
	var buf strings.Builder
	for _, el := range b.ContextElements.Elements {
		fn, ok := contextElementHandlers[el.MixedElementType()]
		if !ok {
			return "", ErrIncorrectBlockType
		}
		s, err := fn(el)
		if err != nil {
			return "", err
		}
		buf.WriteString(s)
	}

	return buf.String(), nil
}

var contextElementHandlers = map[slack.MixedElementType]func(slack.MixedElement) (string, error){
	slack.MixedElementImage: metImage,
}

func metImage(ie slack.MixedElement) (string, error) {
	e, ok := ie.(*slack.ImageBlockElement)
	if !ok {
		return "", ErrIncorrectBlockType
	}
	return fmt.Sprintf(`<img src="%s" alt="%s">`, e.ImageURL, e.AltText), nil
}
