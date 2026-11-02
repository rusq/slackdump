// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package network

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ErrUntrustedFileURL is returned for a file URL that is not served by Slack.
// File download requests authenticate with the user's Slack credentials, so
// callers must not use an archive-controlled URL outside these domains.
var ErrUntrustedFileURL = errors.New("untrusted Slack file download URL")

// IsLocalFileReference reports whether ref is a non-empty reference to a local
// file, such as a dump attachment path, rather than an absolute URL.  It does
// not parse ref, so paths with "%" or Windows drive letters are classified
// correctly.
func IsLocalFileReference(ref string) bool {
	return ref != "" && !strings.Contains(ref, "://")
}

// ValidateSlackFileURL reports whether rawURL is an HTTPS URL served by the
// Slack or GovSlack file hosts.  It is intended for archive-controlled URLs.
// Rejection errors never expose the URL or any part of it.
func ValidateSlackFileURL(rawURL string) error {
	return validateURL(rawURL, func(host string) bool {
		return host == "files.slack.com" || host == "files.slack-gov.com"
	})
}

// ValidateSlackRedirectURL reports whether rawURL is an HTTPS URL on a Slack or
// GovSlack host.  It is intended for redirects issued by Slack itself, which
// may point to workspace or enterprise hosts.
func ValidateSlackRedirectURL(rawURL string) error {
	return validateURL(rawURL, func(host string) bool {
		return host == "slack.com" || host == "slack-gov.com" ||
			strings.HasSuffix(host, ".slack.com") || strings.HasSuffix(host, ".slack-gov.com")
	})
}

func validateURL(rawURL string, hostOK func(string) bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: malformed URL", ErrUntrustedFileURL)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("%w: non-HTTPS scheme", ErrUntrustedFileURL)
	}
	if u.User != nil {
		return fmt.Errorf("%w: userinfo", ErrUntrustedFileURL)
	}
	if !hostOK(strings.ToLower(u.Hostname())) {
		return fmt.Errorf("%w: untrusted host", ErrUntrustedFileURL)
	}
	if port := u.Port(); (port != "" && port != "443") || strings.HasSuffix(u.Host, ":") {
		return fmt.Errorf("%w: unsupported port", ErrUntrustedFileURL)
	}
	return nil
}
