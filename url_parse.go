package slackdump

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type URLInfo struct {
	Channel string
	Thread  int64
}

func (u URLInfo) IsThread() bool {
	return u.Thread > 0
}

func (u URLInfo) IsValid() bool {
	return u.Channel != "" || (u.Channel != "" && u.Thread > 0)
}

var errUnsupportedURL = errors.New("unsuuported URL type")

func ParseURL(slackURL string) (*URLInfo, error) {
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
	if !strings.EqualFold(parts[0], "archives") {
		return nil, errUnsupportedURL
	}

	var ui URLInfo
	switch len(parts) {
	case 3:
		//thread
		if len(parts[2]) == 0 || parts[2][0] != 'p' {
			return nil, errUnsupportedURL
		}
		thread, err := strconv.ParseInt(parts[2][1:], 10, 64)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		ui.Thread = thread
		fallthrough
	case 2:
		// channel
		ui.Channel = parts[1]
	default:
		return nil, errUnsupportedURL
	}
	return &ui, nil
}
