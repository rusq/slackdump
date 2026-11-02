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

// FileURLRejectReason identifies why a URL was not safe for an authenticated
// file download. It deliberately carries no URL data, since archive URLs can
// contain sensitive query parameters.
type FileURLRejectReason string

const (
	FileURLMalformed       FileURLRejectReason = "malformed URL"
	FileURLLocalReference  FileURLRejectReason = "local reference"
	FileURLNonHTTPS        FileURLRejectReason = "non-HTTPS scheme"
	FileURLUserinfo        FileURLRejectReason = "userinfo"
	FileURLUntrustedHost   FileURLRejectReason = "untrusted host"
	FileURLUnsupportedPort FileURLRejectReason = "unsupported port"
)

// FileURLRejectedError reports a safe reason for rejecting an archive URL.
type FileURLRejectedError struct {
	Reason FileURLRejectReason
}

func (e *FileURLRejectedError) Error() string { return string(e.Reason) }

func (e *FileURLRejectedError) Unwrap() error { return ErrUntrustedFileURL }

func rejectFileURL(reason FileURLRejectReason) error {
	return &FileURLRejectedError{Reason: reason}
}

var slackFileHosts = map[string]struct{}{
	"files.slack.com":         {},
	"files.slack-gov.com":     {},
	"files.slack.com:443":     {},
	"files.slack-gov.com:443": {},
}

// ValidateSlackFileURL reports whether rawURL is an HTTPS URL served by the
// Slack or GovSlack file hosts. It permits the default HTTPS port and explicit
// port 443. Rejection errors never expose the URL or any part of it.
func ValidateSlackFileURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rejectFileURL(FileURLMalformed)
	}
	if u.Scheme == "" && u.Host == "" {
		return rejectFileURL(FileURLLocalReference)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return rejectFileURL(FileURLNonHTTPS)
	}
	if u.User != nil {
		return rejectFileURL(FileURLUserinfo)
	}
	if u.Host == "" {
		return rejectFileURL(FileURLUntrustedHost)
	}
	host := strings.ToLower(u.Host)
	if _, ok := slackFileHosts[host]; ok {
		return nil
	}
	if strings.HasPrefix(host, "files.slack.com:") || strings.HasPrefix(host, "files.slack-gov.com:") {
		return rejectFileURL(FileURLUnsupportedPort)
	}
	return rejectFileURL(FileURLUntrustedHost)
}
