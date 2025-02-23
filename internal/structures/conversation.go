package structures

import "github.com/rusq/slack"

func IsThreadStart(m *slack.Message) bool {
	return m.Timestamp == m.ThreadTimestamp
}

func IsEmptyThread(m *slack.Message) bool {
	return m.LatestReply == LatestReplyNoReplies
}

const (
	CUnknown = iota
	CIM      // IM
	CMPIM    // Group IM
	CPrivate // Private Channel
	CPublic  // Public Channel
)

func ChannelType(ch slack.Channel) int {
	switch {
	case ch.IsIM:
		return CIM
	case ch.IsMpIM:
		return CMPIM
	case ch.IsPrivate:
		return CPrivate
	default:
		return CPublic
	}
}
