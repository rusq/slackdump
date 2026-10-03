package renderer

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

var hexColor = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

func escape(s string) string { return html.EscapeString(s) }

// safeURL accepts only links that cannot execute script in a browser. Relative
// viewer routes are kept so static and live renderers can share block output.
func safeURL(raw string, image bool) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	if u.Scheme == "" {
		return raw, strings.HasPrefix(raw, "/") || !strings.HasPrefix(raw, "//")
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return raw, true
	case "mailto":
		return raw, !image
	default:
		return "", false
	}
}

func safeColor(s string) (string, bool) {
	s = strings.TrimPrefix(s, "#")
	return s, hexColor.MatchString(s)
}
