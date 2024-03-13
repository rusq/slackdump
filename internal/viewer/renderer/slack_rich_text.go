package renderer

import (
	"fmt"
	"strings"

	"github.com/rusq/slack"
)

var rteTypeHandlers = map[slack.RichTextElementType]func(slack.RichTextElement) (string, error){
	slack.RTESection: rteSection,
}

func mbtRichText(ib slack.Block) (string, error) {
	b, ok := ib.(*slack.RichTextBlock)
	if !ok {
		return "", ErrIncorrectBlockType
	}
	var buf strings.Builder
	for _, el := range b.Elements {
		fn, ok := rteTypeHandlers[el.RichTextElementType()]
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

func rteSection(ie slack.RichTextElement) (string, error) {
	e, ok := ie.(slack.RichTextSection)
	if !ok {
		return "", ErrIncorrectBlockType
	}
	var buf strings.Builder
	for _, el := range e.Elements {
		fn, ok := rtseHandlers[el.RichTextSectionElementType()]
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

var rtseHandlers = map[slack.RichTextSectionElementType]func(slack.RichTextSectionElement) (string, error){
	slack.RTSEText: rtseText,
}

func rtseText(ie slack.RichTextSectionElement) (string, error) {
	e, ok := ie.(*slack.RichTextSectionTextElement)
	if !ok {
		return "", fmt.Errorf("%T: %w", ie, ErrIncorrectBlockType)
	}
	var t = e.Text
	if e.Style == nil {
		return t, nil
	}
	if e.Style.Bold {
		t = fmt.Sprintf("<b>%s</b>", t)
	}
	if e.Style.Italic {
		t = fmt.Sprintf("<i>%s</i>", t)
	}
	if e.Style.Strike {
		t = fmt.Sprintf("<s>%s</s>", t)
	}
	if e.Style.Code {
		t = fmt.Sprintf("<code>%s</code>", t)
	}
	return t, nil
}
