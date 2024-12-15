// Package structures provides functions to parse Slack data types.
package structures

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
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

var ErrInvalidDomain = errors.New("invalid domain")

// ExtractWorkspace takes a workspace name or URL and returns the workspace name.
func ExtractWorkspace(workspace string) (string, error) {
	if !strings.Contains(workspace, ".") {
		return workspace, nil
	}
	if strings.HasPrefix(workspace, "https://") {
		uri, err := url.Parse(workspace)
		if err != nil {
			return "", err
		}
		workspace = uri.Host
	}
	if !strings.Contains(workspace, ".slack.com") {
		return "", ErrInvalidDomain
	}

	parts := strings.Split(workspace, ".")
	switch len(parts) {
	case 3, 4:
		return parts[0], nil
	default:
		return "", fmt.Errorf("invalid workspace: %s", workspace)
	}
}

// NVLTime returns the default time if the given time is zero.
func NVLTime(t time.Time, def time.Time) time.Time {
	if t.IsZero() {
		return def
	}
	return t
}
