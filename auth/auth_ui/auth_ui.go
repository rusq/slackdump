package auth_ui

import (
	"errors"
	"fmt"
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
	// LUserBrowser is the google auth option
	LUserBrowser
	// LCancel should be returned if the user cancels the login intent.
	LCancel
)

var ErrInvalidDomain = errors.New("invalid domain")

// Sanitize takes a workspace name or URL and returns the workspace name.
func Sanitize(workspace string) (string, error) {
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
