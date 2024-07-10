package renderer

import "html"

func parseSlackMd(s string) string {
	// TODO parse legacy markdown
	return "<pre>" + html.EscapeString(s) + "</pre>"
}
