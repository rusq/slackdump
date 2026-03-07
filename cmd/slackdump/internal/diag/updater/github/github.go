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

// Package github provides a very limited github API client with optional authentication.
//
// # Rate Limiting
//
// GitHub API has strict rate limits:
//   - Anonymous requests: 60 requests/hour per IP
//   - Authenticated requests: 5,000 requests/hour per user
//
// To use authentication, set the GITHUB_TOKEN environment variable with a
// Personal Access Token (PAT). The token does not require any specific scopes
// for accessing public release information.
//
// Rate limit information is logged when available in API responses via the
// X-RateLimit-* headers.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime/trace"
	"strconv"
	"time"
)

const (
	scheme           = "https"
	githubAPI        = "api.github.com"
	githubAPIVersion = "2022-11-28"
	contentType      = "application/vnd.github+json"

	hdrAccept           = "Accept"
	hdrGitHubAPIVersion = "X-GitHub-Api-Version"
	hdrAuthorization    = "Authorization"
	hdrRateLimit        = "X-RateLimit-Limit"
	hdrRateLimitUsed    = "X-RateLimit-Used"
	hdrRateLimitRemain  = "X-RateLimit-Remaining"
	hdrRateLimitReset   = "X-RateLimit-Reset"

	// Environment variable for GitHub token
	envGitHubToken = "GITHUB_TOKEN"
)

// Client is a very limited Github client with optional authentication.
// It supports both anonymous and authenticated requests to the GitHub API.
//
// For authenticated requests, set the GITHUB_TOKEN environment variable.
// This increases the rate limit from 60 to 5,000 requests per hour.
type Client struct {
	// Owner is the Github owner.
	Owner string
	// Repo is the Github repository.
	Repo string
	// Prerelease indicates if we want to accept pre-release versions.
	Prerelease bool
	// Token is the optional GitHub Personal Access Token for authentication.
	// If empty, requests will be anonymous.
	Token string
}

var ErrNoReleases = errors.New("no releases")

// NewClient creates a new GitHub client.
// It automatically reads the GITHUB_TOKEN environment variable for authentication.
func NewClient(owner, repo string, prerelease bool) *Client {
	return &Client{
		Owner:      owner,
		Repo:       repo,
		Prerelease: prerelease,
		Token:      os.Getenv(envGitHubToken),
	}
}

func (cl Client) Latest(ctx context.Context) (*Release, error) {
	ctx, task := trace.NewTask(ctx, "Latest")
	defer task.End()

	var params = url.Values{
		"per_page": []string{"1"},
	}
	uri := cl.url(path.Join("repos", cl.Owner, cl.Repo, "releases", "latest"), params)
	slog.DebugContext(ctx, "Rendered", "uri", uri)

	resp, err := cl.get(ctx, uri)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "Failed to close response body", "err", err)
		}
	}()

	r, err := cl.parseRelease(resp)
	if err != nil {
		return nil, fmt.Errorf("unable to find latest release: %w", err)
	}

	return r, nil
}

func (cl Client) parseRelease(resp *http.Response) (*Release, error) {
	var r Release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("unable to parse Github releases: %w", err)
	}
	if r.Draft {
		return nil, ErrNoReleases
	}
	if !cl.Prerelease && r.Prerelease {
		return nil, ErrNoReleases
	}
	return &r, nil
}

func (cl Client) ByTag(ctx context.Context, tag string) (*Release, error) {
	uri := cl.url(path.Join("repos", cl.Owner, cl.Repo, "releases", "tags", tag), nil)
	slog.DebugContext(ctx, "Rendered", "uri", uri)

	resp, err := cl.get(ctx, uri)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "Failed to close response body", "err", err)
		}
	}()

	r, err := cl.parseRelease(resp)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Github release %q: %w", tag, err)
	}

	return r, nil
}

func (cl Client) url(path string, params url.Values) string {
	var u = url.URL{
		Scheme:   scheme,
		Host:     githubAPI,
		Path:     path,
		RawQuery: params.Encode(),
	}
	return u.String()
}

func (cl Client) get(ctx context.Context, uri string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	return cl.do(req)
}

func (cl Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Add(hdrAccept, contentType)
	req.Header.Add(hdrGitHubAPIVersion, githubAPIVersion)

	// Add authentication if token is available
	if cl.Token != "" {
		req.Header.Add(hdrAuthorization, "Bearer "+cl.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to issue Github API request: %w", err)
	}

	// Log rate limit information if available
	cl.logRateLimitInfo(req.Context(), resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid API status code (%d)", resp.StatusCode)
	}
	return resp, nil
}

// logRateLimitInfo logs GitHub API rate limit information from response headers.
func (cl Client) logRateLimitInfo(ctx context.Context, resp *http.Response) {
	limit := resp.Header.Get(hdrRateLimit)
	used := resp.Header.Get(hdrRateLimitUsed)
	remaining := resp.Header.Get(hdrRateLimitRemain)
	resetStr := resp.Header.Get(hdrRateLimitReset)

	// Only log if we have rate limit headers
	if limit == "" {
		return
	}

	attrs := []any{
		"limit", limit,
		"used", used,
		"remaining", remaining,
	}

	// Parse reset time if available
	if resetStr != "" {
		if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			resetTime := time.Unix(resetUnix, 0)
			attrs = append(attrs, "reset_at", resetTime.Format(time.RFC3339))
		}
	}

	// Log authentication status
	authStatus := "anonymous"
	if cl.Token != "" {
		authStatus = "authenticated"
	}
	attrs = append(attrs, "auth", authStatus)

	// Warn if we're running low on rate limit
	if remainingInt, err := strconv.Atoi(remaining); err == nil {
		if remainingInt < 10 {
			slog.WarnContext(ctx, "GitHub API rate limit running low", attrs...)
		} else {
			slog.DebugContext(ctx, "GitHub API rate limit status", attrs...)
		}
	} else {
		slog.DebugContext(ctx, "GitHub API rate limit status", attrs...)
	}
}
