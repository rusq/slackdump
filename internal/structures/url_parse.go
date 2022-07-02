package structures

// In this file: slack URL parsing functions.

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

const linkSep = ":"

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
	return strings.Join([]string{u.Channel, u.ThreadTS}, linkSep)
}

func ParseLink(link string) (SlackLink, error) {
	if IsURL(link) {
		sl, err := ParseURL(link)
		if err != nil {
			return SlackLink{}, err
		}
		return *sl, nil
	}

	id, ts, _ := strings.Cut(link, linkSep)
	return SlackLink{Channel: id, ThreadTS: ts}, nil
}

var (
	ErrUnsupportedURL = errors.New("unsupported URL type")
	ErrNoURL          = errors.New("no url provided")
	ErrNotSlackURL    = errors.New("not a slack URL")
)

func ParseURL(slackURL string) (*SlackLink, error) {
	if slackURL == "" {
		return nil, ErrNoURL
	}
	if !IsValidSlackURL(slackURL) {
		return nil, ErrNotSlackURL
	}
	uri, err := url.Parse(slackURL)
	if err != nil {
		return nil, errors.WithStack(err)
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
var slackURLRe = regexp.MustCompile(`^https:\/\/[\w]+\.slack\.com\/archives\/[A-Z]{1}[A-Z0-9]+(\/p(\d+))?$`)

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
