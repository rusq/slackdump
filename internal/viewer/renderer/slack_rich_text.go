package renderer

import (
	"fmt"
	"log/slog"
	"strings"

	emj "github.com/enescakir/emoji"
	"github.com/rusq/slack"
)

func (s *Slack) mbtRichText(ib slack.Block) (string, error) {
	b, ok := ib.(*slack.RichTextBlock)
	if !ok {
		return "", NewErrIncorrectType(&slack.RichTextBlock{}, ib)
	}
	var buf strings.Builder
	for _, el := range b.Elements {
		fn, ok := rteTypeHandlers[el.RichTextElementType()]
		if !ok {
			return "", NewErrMissingHandler(el.RichTextElementType())
		}
		s, err := fn(s, el)
		if err != nil {
			return "", err
		}
		buf.WriteString(s)
	}

	return buf.String(), nil
}

func (s *Slack) rteSection(ie slack.RichTextElement) (string, error) {
	e, ok := ie.(*slack.RichTextSection)
	if !ok {
		return "", NewErrIncorrectType(&slack.RichTextSection{}, ie)
	}
	var buf strings.Builder
	for _, el := range e.Elements {
		fn, ok := rtseTypeHandlers[el.RichTextSectionElementType()]
		if !ok {
			return "", NewErrMissingHandler(el.RichTextSectionElementType())
		}
		s, err := fn(s, el)
		if err != nil {
			return "", err
		}
		buf.WriteString(s)
	}

	return buf.String(), nil
}

func (s *Slack) rtseText(ie slack.RichTextSectionElement) (string, error) {
	e, ok := ie.(*slack.RichTextSectionTextElement)
	if !ok {
		return "", NewErrIncorrectType(&slack.RichTextSectionTextElement{}, ie)
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

func (s *Slack) rtseLink(ie slack.RichTextSectionElement) (string, error) {
	e, ok := ie.(*slack.RichTextSectionLinkElement)
	if !ok {
		return "", NewErrIncorrectType(&slack.RichTextSectionLinkElement{}, ie)
	}
	if e.Text == "" {
		e.Text = e.URL
	}
	return fmt.Sprintf("<a href=\"%s\">%s</a>", e.URL, e.Text), nil
}

func (s *Slack) rteList(ie slack.RichTextElement) (string, error) {
	e, ok := ie.(*slack.RichTextList)
	if !ok {
		return "", NewErrIncorrectType(&slack.RichTextList{}, ie)
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
			return "", NewErrMissingHandler(el.RichTextElementType())
		}
		s, err := fn(s, el)
		if err != nil {
			return "", err
		}
		buf.WriteString(fmt.Sprintf("<li>%s</li>", s))
	}
	buf.WriteString(tgClose)
	return buf.String(), nil
}

func (s *Slack) rteQuote(ie slack.RichTextElement) (string, error) {
	e, ok := ie.(*slack.RichTextQuote)
	if !ok {
		return "", NewErrIncorrectType(&slack.RichTextQuote{}, ie)
	}
	var buf strings.Builder
	buf.WriteString("<blockquote>")
	for _, el := range e.Elements {
		fn, ok := rtseTypeHandlers[el.RichTextSectionElementType()]
		if !ok {
			return "", NewErrMissingHandler(el.RichTextSectionElementType())
		}
		s, err := fn(s, el)
		if err != nil {
			return "", err
		}
		buf.WriteString(s)
	}
	buf.WriteString("</blockquote>")
	return buf.String(), nil
}

func (s *Slack) rtePreformatted(ie slack.RichTextElement) (string, error) {
	e, ok := ie.(*slack.RichTextPreformatted)
	if !ok {
		return "", NewErrIncorrectType(&slack.RichTextPreformatted{}, ie)
	}
	var buf strings.Builder
	buf.WriteString("<pre>")
	for _, el := range e.Elements {
		fn, ok := rtseTypeHandlers[el.RichTextSectionElementType()]
		if !ok {
			return "", NewErrMissingHandler(el.RichTextSectionElementType())
		}
		s, err := fn(s, el)
		if err != nil {
			return "", err
		}
		buf.WriteString(s)
	}
	buf.WriteString("</pre>")
	return buf.String(), nil
}

func (s *Slack) rtseUser(ie slack.RichTextSectionElement) (string, error) {
	e, ok := ie.(*slack.RichTextSectionUserElement)
	if !ok {
		return "", NewErrIncorrectType(&slack.RichTextSectionUserElement{}, ie)
	}
	var name string

	if u, ok := s.uu[e.UserID]; s.uu != nil && ok {
		name = u.Name
	} else {
		slog.Warn("user not found", "user_id", e.UserID, "user", u)
		name = e.UserID
	}

	// TODO: link user.
	return applyStyle(fmt.Sprintf("<@%s>", name), e.Style), nil
}

func (s *Slack) rtseEmoji(ie slack.RichTextSectionElement) (string, error) {
	e, ok := ie.(*slack.RichTextSectionEmojiElement)
	if !ok {
		return "", NewErrIncorrectType(&slack.RichTextSectionEmojiElement{}, ie)
	}
	// TODO: resolve and render emoji.
	em := emj.Parse(fmt.Sprintf(":%s:", e.Name))
	return applyStyle(em, e.Style), nil
}

func (s *Slack) rtseChannel(ie slack.RichTextSectionElement) (string, error) {
	e, ok := ie.(*slack.RichTextSectionChannelElement)
	if !ok {
		return "", NewErrIncorrectType(&slack.RichTextSectionChannelElement{}, ie)
	}
	var name string
	if c, ok := s.cc[e.ChannelID]; s.cc != nil && ok {
		name = c.Name
	} else {
		slog.Warn("channel not found", "channel_id", e.ChannelID)
		name = e.ChannelID
	}

	return div("slack-rtse-channel", applyStyle(fmt.Sprintf("<#%s>", name), e.Style)), nil
}
