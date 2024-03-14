package renderer

import (
	"fmt"
	"strings"

	"github.com/rusq/slack"
)

func mbtContext(ib slack.Block) (string, error) {
	b, ok := ib.(*slack.ContextBlock)
	if !ok {
		return "", NewErrIncorrectType(&slack.ContextBlock{}, ib)
	}
	var buf strings.Builder
	for _, el := range b.ContextElements.Elements {
		fn, ok := contextElementHandlers[el.MixedElementType()]
		if !ok {
			return "", NewErrMissingHandler(el.MixedElementType())
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
	slack.MixedElementText:  metText,
}

func metImage(ie slack.MixedElement) (string, error) {
	e, ok := ie.(*slack.ImageBlockElement)
	if !ok {
		return "", NewErrIncorrectType(&slack.ImageBlockElement{}, ie)
	}
	return fmt.Sprintf(`<img src="%s" alt="%s">`, e.ImageURL, e.AltText), nil
}

func metText(ie slack.MixedElement) (string, error) {
	e, ok := ie.(*slack.TextBlockObject)
	if !ok {
		return "", NewErrIncorrectType(&slack.TextBlockObject{}, ie)
	}
	return e.Text, nil
}
