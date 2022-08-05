package structures

// In this file: slack URL parsing functions.

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"errors"
)

const linkSep = ":"

var (
	ErrUnsupportedURL = errors.New("unsupported URL type")
	ErrNoURL          = errors.New("no url provided")
	ErrNotSlackURL    = errors.New("not a slack URL")
	ErrInvalidLink    = errors.New("invalid link")
)

type SlackLink struct {
	Channel  string
	ThreadTS string
}

func (u SlackLink) IsThread() bool {
	if !u.IsValid() {
		return false
	}
	return u.ThreadTS != ""
}

func (u SlackLink) IsValid() bool {
	return u.Channel != "" || (u.Channel != "" && u.ThreadTS != "")
}

func (u SlackLink) String() string {
	if !u.IsValid() {
		return "<invalid slack link>"
	}
	if !u.IsThread() {
		return u.Channel
	}
	return strings.Join([]string{u.Channel, u.ThreadTS}, linkSep)
}

var linkRe = regexp.MustCompile(`^[A-Za-z]{1}[A-Za-z0-9]+(:[0-9]+\.[0-9]+)?$`)

// ParseLink parses the slack link string.  It supports the following formats:
//
//   - XXXXXXX                   - channel ID
//   - XXXXXXX:99999999.99999    - channel ID and thread ID
//   - https://<valid slack URL> - slack URL link.
// It returns the SlackLink or error.
func ParseLink(link string) (SlackLink, error) {
	if IsURL(link) {
		sl, err := ParseURL(link)
		if err != nil {
			return SlackLink{}, err
		}
		return *sl, nil
	}
	if !linkRe.MatchString(link) {
		return SlackLink{}, fmt.Errorf("%w: %q", ErrInvalidLink, link)
	}

	id, ts, _ := strings.Cut(link, linkSep)
	return SlackLink{Channel: id, ThreadTS: ts}, nil
}

// ParseURL parses the slack link in the format of
// https://xxxx.slack.com/archives/XXXXX[/p99999999]
func ParseURL(slackURL string) (*SlackLink, error) {
	if slackURL == "" {
		return nil, ErrNoURL
	}
	if !IsValidSlackURL(slackURL) {
		return nil, ErrNotSlackURL
	}
	uri, err := url.Parse(slackURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL %q: %w", slackURL, err)
	}
	if len(uri.Path) == 0 {
		return nil, errors.New("url should point to a DM or Public conversation or a Slack Thread")
	}

	parts := strings.Split(strings.TrimPrefix(uri.Path, "/"), "/")
	if !strings.EqualFold(parts[0], "archives") || len(parts) < 2 || parts[1] == "" {
		return nil, ErrUnsupportedURL
	}

	var ui SlackLink
	switch len(parts) {
	case 3:
		//thread
		ts, err := ParseThreadID(parts[2])
		if err != nil {
			return nil, ErrUnsupportedURL
		}
		ui.ThreadTS = FormatSlackTS(ts)
		fallthrough
	case 2:
		// channel
		ui.Channel = parts[1]
	default:
		return nil, ErrUnsupportedURL
	}
	if !ui.IsValid() {
		return nil, fmt.Errorf("invalid URL: %q", slackURL)
	}
	return &ui, nil
}

// Sample: https://ora600.slack.com/archives/CHM82GF99/p1577694990000400
//
// > Your workspace URL can only contain lowercase letters, numbers and dashes
// > (and must start with a letter or number).
var slackURLRe = regexp.MustCompile(`^https:\/\/[a-zA-Z0-9]{1}[-\w]+\.slack\.com\/archives\/[A-Z]{1}[A-Z0-9]+(\/p(\d+))?$`)

// IsValidSlackURL returns true if the value looks like valid Slack URL, false
// if not.
func IsValidSlackURL(s string) bool {
	return slackURLRe.MatchString(s)
}

func IsURL(s string) bool {
	return strings.HasPrefix(strings.ToLower(s), "https://")
}

// ResolveURLs normalises all channels to ID form.  If the idsOrURLs[i] is
// a channel ID, it is unmodified, if it is URL - it is parsed and replaced
// with the channel ID.  If the channel is marked for exclusion in the list
// it will retain this status.
func ResolveURLs(idsOrURLs []string) ([]string, error) {
	var ret = make([]string, len(idsOrURLs))
	for i, val := range idsOrURLs {
		if val == "" {
			continue
		}

		restorePrefix := HasExcludePrefix(val)
		if restorePrefix {
			val = val[len(excludePrefix):] // remove exclude prefix for the sake of parsing
		}

		if !IsValidSlackURL(val) {
			continue
		}
		ch, err := ParseURL(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing Slack URL %q: %w", val, err)
		}
		if !ch.IsValid() {
			return nil, fmt.Errorf("not a valid Slack URL: %s", val)
		}

		if restorePrefix {
			// restoring exclude prefix
			ret[i] = excludePrefix + ch.Channel
		} else {
			ret[i] = ch.Channel
		}
	}
	return ret, nil
}
