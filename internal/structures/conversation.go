package structures

import "github.com/rusq/slack"

// IsThreadStart check if the message is a lead message of a thread.
func IsThreadStart(m *slack.Message) bool {
	return m.Timestamp == m.ThreadTimestamp
}

// IsEmptyThread checks if the message is a thread with no replies.
func IsEmptyThread(m *slack.Message) bool {
	return m.LatestReply == LatestReplyNoReplies
}

// IsThreadMessage checks if the message is a thread message (not lead).
func IsThreadMessage(m *slack.Msg) bool {
	return m.ThreadTimestamp != "" && m.ThreadTimestamp != m.Timestamp
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
