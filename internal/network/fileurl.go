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
	"net/url"
	"strings"
)

// ErrUntrustedFileURL is returned for a file URL that is not served by Slack.
// File download requests authenticate with the user's Slack credentials, so
// callers must not use an archive-controlled URL outside these domains.
var ErrUntrustedFileURL = errors.New("untrusted Slack file download URL")

var slackFileDomains = []string{"slack.com", "slack-gov.com"}

// ValidateSlackFileURL reports whether rawURL is an HTTPS URL served by Slack
// or GovSlack. It permits the default HTTPS port and explicit port 443.
func ValidateSlackFileURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || !strings.EqualFold(u.Scheme, "https") || u.Host == "" || u.User != nil {
		return ErrUntrustedFileURL
	}
	if port := u.Port(); port != "" && port != "443" {
		return ErrUntrustedFileURL
	}

	host := strings.ToLower(u.Hostname())
	for _, domain := range slackFileDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return nil
		}
	}
	return ErrUntrustedFileURL
}
