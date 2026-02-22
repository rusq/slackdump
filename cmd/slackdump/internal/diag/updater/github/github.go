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

// Package github provides a very limited anonymous github API client.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"runtime/trace"
)

const (
	scheme           = "https"
	githubAPI        = "api.github.com"
	githubAPIVersion = "2022-11-28"
	contentType      = "application/vnd.github+json"

	hdrAccept           = "Accept"
	hdrGitHubAPIVersion = "X-GitHub-Api-Version"
)

// Client is a very limited Github client.
type Client struct {
	// Owner is the Github owner.
	Owner string
	// Repo is the Github repository.
	Repo string
	// Prerelease indicates if we want to accept pre-release versions.
	Prerelease bool
}

var ErrNoReleases = errors.New("no releases")

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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to issue Github API request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid API status code (%d)", resp.StatusCode)
	}
	return resp, nil
}
