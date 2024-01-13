package auth_ui

import (
	"net/url"
	"strings"
)

// LoginType is the login type, that is used to choose the authentication flow,
// for example login headlessly or interactively.
type LoginType int8

const (
	// LInteractive is the SSO login type (Google, Apple, etc).
	LInteractive LoginType = iota
	// LHeadless is the email/password login type.
	LHeadless
	// LCancel should be returned if the user cancels the login intent.
	LCancel
)

// Sanitize takes a workspace name or URL and returns the workspace name.
func Sanitize(workspace string) (string, error) {
	if !strings.Contains(workspace, ".slack.com") {
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
	parts := strings.Split(workspace, ".")
	return parts[0], nil
}
