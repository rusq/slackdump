package renderer

import (
	"fmt"
	"strings"

	"github.com/rusq/slack"
)

var rteTypeHandlers = map[slack.RichTextElementType]func(slack.RichTextElement) (string, error){}

func init() {
	rteTypeHandlers[slack.RTESection] = rteSection
	rteTypeHandlers[slack.RTEList] = rteList
	rteTypeHandlers[slack.RTEQuote] = rteQuote
	rteTypeHandlers[slack.RTEPreformatted] = rtePreformatted
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
	e, ok := ie.(*slack.RichTextSection)
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
	slack.RTSEText:  rtseText,
	slack.RTSELink:  rtseLink,
	slack.RTSEUser:  rtseUser,
	slack.RTSEEmoji: rtseEmoji,
}

func rtseText(ie slack.RichTextSectionElement) (string, error) {
	e, ok := ie.(*slack.RichTextSectionTextElement)
	if !ok {
		return "", fmt.Errorf("%T: %w", ie, ErrIncorrectBlockType)
	}
	var t = strings.Replace(e.Text, "\n", "<br>", -1)

	return applyStyle(t, e.Style), nil
}

func applyStyle(s string, style *slack.RichTextSectionTextStyle) string {
	if style == nil {
		return s
	}
	if style.Bold {
		s = fmt.Sprintf("<b>%s</b>", s)
	}
	if style.Italic {
		s = fmt.Sprintf("<i>%s</i>", s)
	}
	if style.Strike {
		s = fmt.Sprintf("<s>%s</s>", s)
	}
	if style.Code {
		s = fmt.Sprintf("<code>%s</code>", s)
	}
	return s
}

func rtseLink(ie slack.RichTextSectionElement) (string, error) {
	e, ok := ie.(*slack.RichTextSectionLinkElement)
	if !ok {
		return "", fmt.Errorf("%T: %w", ie, ErrIncorrectBlockType)
	}
	if e.Text == "" {
		e.Text = e.URL
	}
	return fmt.Sprintf("<a href=\"%s\">%s</a>", e.URL, e.Text), nil
}

func rteList(ie slack.RichTextElement) (string, error) {
	e, ok := ie.(*slack.RichTextList)
	if !ok {
		return "", ErrIncorrectBlockType
	}
	// const orderedTypes = "1aAiI"
	var tgOpen, tgClose string
	if e.Style == slack.RTEListOrdered {
		// TODO: type alternation on even/odd
		// https://www.w3schools.com/tags/att_ol_type.asp
		tgOpen, tgClose = "<ol>", "</ol>"
	} else {
		tgOpen, tgClose = "<ul>", "</ul>"
	}
	tgOpen, tgClose = strings.Repeat(tgOpen, e.Indent+1), strings.Repeat(tgClose, e.Indent+1)
	var buf strings.Builder
	buf.WriteString(tgOpen)
	for _, el := range e.Elements {
		fn, ok := rteTypeHandlers[el.RichTextElementType()]
		if !ok {
			return "", ErrIncorrectBlockType
		}
		s, err := fn(el)
		if err != nil {
			return "", err
		}
		buf.WriteString(fmt.Sprintf("<li>%s</li>", s))
	}
	buf.WriteString(tgClose)
	return buf.String(), nil
}

func rteQuote(ie slack.RichTextElement) (string, error) {
	e, ok := ie.(*slack.RichTextQuote)
	if !ok {
		return "", ErrIncorrectBlockType
	}
	var buf strings.Builder
	buf.WriteString("<blockquote>")
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
	buf.WriteString("</blockquote>")
	return buf.String(), nil
}

func rtePreformatted(ie slack.RichTextElement) (string, error) {
	e, ok := ie.(*slack.RichTextPreformatted)
	if !ok {
		return "", ErrIncorrectBlockType
	}
	var buf strings.Builder
	buf.WriteString("<pre>")
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
	buf.WriteString("</pre>")
	return buf.String(), nil
}

func rtseUser(ie slack.RichTextSectionElement) (string, error) {
	e, ok := ie.(*slack.RichTextSectionUserElement)
	if !ok {
		return "", fmt.Errorf("%T: %w", ie, ErrIncorrectBlockType)
	}
	// TODO: link user.
	return applyStyle(fmt.Sprintf("<@%s>", e.UserID), e.Style), nil
}

func rtseEmoji(ie slack.RichTextSectionElement) (string, error) {
	e, ok := ie.(*slack.RichTextSectionEmojiElement)
	if !ok {
		return "", fmt.Errorf("%T: %w", ie, ErrIncorrectBlockType)
	}
	// TODO: resolve and render emoji.
	return applyStyle(fmt.Sprintf(":%s:", e.Name), e.Style), nil
}
