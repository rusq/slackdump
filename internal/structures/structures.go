// Package structures provides functions to parse Slack data types.
package structures

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
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
	if !strings.Contains(workspace, ".slack.com") && !strings.Contains(workspace, ".") {
		return workspace, nil
	}
	if strings.HasPrefix(workspace, "https://") {
		uri, err := url.Parse(workspace)
		if err != nil {
			return "", err
		}
		workspace = uri.Host
	}
	// parse
	name, domain, found := strings.Cut(workspace, ".")
	if !found {
		return "", errors.New("workspace name is empty")
	}
	if strings.TrimRight(domain, "/") != "slack.com" {
		return "", fmt.Errorf("%s: %w", domain, ErrInvalidDomain)
	}
	return name, nil
}
