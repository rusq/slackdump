package slackdump

// In this file: slack URL parsing functions.

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type urlInfo struct {
	Channel  string
	ThreadTS string
}

func (u urlInfo) IsThread() bool {
	return u.ThreadTS != ""
}

func (u urlInfo) IsValid() bool {
	return u.Channel != "" || (u.Channel != "" && u.ThreadTS != "")
}

var errUnsupportedURL = errors.New("unsuuported URL type")

func parseURL(slackURL string) (*urlInfo, error) {
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
		return nil, errUnsupportedURL
	}

	var ui urlInfo
	switch len(parts) {
	case 3:
		//thread
		ts, err := parseThreadID(parts[2])
		if err != nil {
			return nil, errUnsupportedURL
		}
		ui.ThreadTS = formatSlackTS(ts)
		fallthrough
	case 2:
		// channel
		ui.Channel = parts[1]
	default:
		return nil, errUnsupportedURL
	}
	if !ui.IsValid() {
		return nil, fmt.Errorf("invalid URL: %q", slackURL)
	}
	return &ui, nil
}
