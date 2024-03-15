package renderer

import "github.com/rusq/slack"

var (
	blockTypeHandlers = map[slack.MessageBlockType]func(*Slack, slack.Block) (string, error){
		slack.MBTRichText: (*Slack).mbtRichText,
		slack.MBTImage:    (*Slack).mbtImage,
		slack.MBTContext:  (*Slack).mbtContext,
	}

	blockTypeClass = map[slack.MessageBlockType]string{
		slack.MBTRichText: "slack-rich-text-block",
		slack.MBTImage:    "slack-image-block",
		slack.MBTContext:  "slack-context-block",
	}
)

// rte - rich text element
var (
	rteTypeHandlers = map[slack.RichTextElementType]func(*Slack, slack.RichTextElement) (string, error){}

	rteTypeClass = map[slack.RichTextElementType]string{
		slack.RTESection:      "slack-rich-text-section",
		slack.RTEList:         "slack-rich-text-list",
		slack.RTEQuote:        "slack-rich-text-quote",
		slack.RTEPreformatted: "slack-rich-text-preformatted",
	}
)

func init() {
	rteTypeHandlers[slack.RTESection] = (*Slack).rteSection
	rteTypeHandlers[slack.RTEList] = (*Slack).rteList
	rteTypeHandlers[slack.RTEQuote] = (*Slack).rteQuote
	rteTypeHandlers[slack.RTEPreformatted] = (*Slack).rtePreformatted
}

// rtse - rich text section element
var (
	rtseTypeHandlers = map[slack.RichTextSectionElementType]func(*Slack, slack.RichTextSectionElement) (string, error){
		slack.RTSEText:    (*Slack).rtseText,
		slack.RTSELink:    (*Slack).rtseLink,
		slack.RTSEUser:    (*Slack).rtseUser,
		slack.RTSEEmoji:   (*Slack).rtseEmoji,
		slack.RTSEChannel: (*Slack).rtseChannel,
	}

	rtseTypeClass = map[slack.RichTextSectionElementType]string{
		slack.RTSEText:    "slack-rich-text-section-text",
		slack.RTSELink:    "slack-rich-text-section-link",
		slack.RTSEUser:    "slack-rich-text-section-user",
		slack.RTSEEmoji:   "slack-rich-text-section-emoji",
		slack.RTSEChannel: "slack-rich-text-section-channel",
	}
)
