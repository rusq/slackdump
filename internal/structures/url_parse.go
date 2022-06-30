package structures

// In this file: slack URL parsing functions.

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type UrlInfo struct {
	Channel  string
	ThreadTS string
}

func (u UrlInfo) IsThread() bool {
	return u.ThreadTS != ""
}

func (u UrlInfo) IsValid() bool {
	return u.Channel != "" || (u.Channel != "" && u.ThreadTS != "")
}

var ErrUnsupportedURL = errors.New("unsuuported URL type")

func ParseURL(slackURL string) (*UrlInfo, error) {
	if slackURL == "" {
		return nil, errors.New("no url provided")
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

	var ui UrlInfo
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

// IsURL returns true if the value looks like URL, false if not.
func IsURL(s string) bool {
	return strings.HasPrefix(strings.ToLower(s), "https://")
}
