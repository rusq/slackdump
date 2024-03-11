package structures

import "github.com/rusq/slack"

func IsThreadStart(m *slack.Message) bool {
	return m.Timestamp == m.ThreadTimestamp
}

const (
	CUnknown = iota
	CIM
	CMPIM
	CPrivate
	CPublic
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
