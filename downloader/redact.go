package downloader

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// safeURLLabel intentionally exposes only the path's final component. Download
// URLs commonly contain bearer tokens in query parameters or userinfo.
func safeURLLabel(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "invalid-url"
	}
	if name := path.Base(u.Path); name != "." && name != "/" && name != "" {
		return name
	}
	return "download"
}

type redactedError struct{ err error }

func (e redactedError) Error() string { return redactURLText(e.err.Error()) }
func (e redactedError) Unwrap() error { return e.err }
func redactError(err error) error {
	if err == nil {
		return nil
	}
	return redactedError{err: err}
}

func redactURLText(s string) string {
	for _, word := range strings.Fields(s) {
		prefix, rawURL, suffix := splitURLToken(word)
		u, err := url.Parse(rawURL)
		if err == nil && u.Scheme != "" && (u.User != nil || u.RawQuery != "" || u.Fragment != "") {
			s = strings.Replace(s, word, prefix+fmt.Sprintf("%s://%s%s", u.Scheme, u.Hostname(), u.EscapedPath())+suffix, 1)
		}
	}
	return s
}

// splitURLToken retains punctuation that error formatters place around URLs,
// such as the quotes and colon in url.Error's Get "https://...": form.
func splitURLToken(word string) (prefix, rawURL, suffix string) {
	for len(word) > 0 && strings.ContainsRune("\"'(<[", rune(word[0])) {
		prefix += word[:1]
		word = word[1:]
	}
	for len(word) > 0 && strings.ContainsRune("\"')]>.,:;", rune(word[len(word)-1])) {
		suffix = word[len(word)-1:] + suffix
		word = word[:len(word)-1]
	}
	return prefix, word, suffix
}
