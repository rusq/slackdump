// Package structures provides functions to parse Slack data types.
package structures

import (
	"errors"
	"regexp"
)

const (
	LatestReplyNoReplies = "0000000000.000000"
)

const (
	SubTypeThreadBroadcast = "thread_broadcast"
)

// tokenRe is a loose regular expression to match Slack API tokens.
// a - app, b - bot, c - client, e - export, p - legacy
var tokenRE = regexp.MustCompile(`xox[abcep]-[0-9]+-[0-9]+-[0-9]+-[0-9a-f]{64}`)

var errInvalidToken = errors.New("token must start with xoxa-, xoxb-, xoxc-, xoxe- or xoxp- and be followed by 3 group of numbers and then 64 hexadecimal characters")

func ValidateToken(token string) error {
	if !tokenRE.MatchString(token) {
		return errInvalidToken
	}
	return nil
}
