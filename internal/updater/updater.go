// Package updater provides online update and version checking functions.
package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/mod/semver"
)

type Updater struct {
	ghReleaseURL string
}

const numReleases = 1

func NewUpdater() *Updater {
	return &Updater{
		ghReleaseURL: fmt.Sprintf("https://api.github.com/repos/rusq/slackdump/releases?per_page=%d", numReleases),
	}
}

var (
	ErrStatus        = errors.New("invalid status code")
	ErrNoVersions    = errors.New("no versions found")
	ErrNoNewReleases = errors.New("no new releases")
)

type Version struct {
	Version    string
	ReleasedAt time.Time
	Notes      string
	IsStable   bool
}

// Latest returns the latest version released on github
func (u *Updater) Latest(ctx context.Context) (Version, error) {
	grr, err := u.getLatestRelease(ctx)
	if err != nil {
		return Version{}, err
	}
	v := Version{
		Version:    grr.TagName,
		ReleasedAt: grr.PublishedAt,
		Notes:      grr.Body,
		IsStable:   !grr.PreRelease,
	}
	if grr.Draft {
		return v, ErrNoNewReleases
	}
	return v, nil
}

type ghReleaseResponse struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
	PreRelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
}

// getLatestRelease returns the latest release from the Github, assuming
// that uri is the Releases URL.
func (u *Updater) getLatestRelease(ctx context.Context) (*ghReleaseResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.ghReleaseURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: want 200, got %d", ErrStatus, resp.StatusCode)
	}
	var versions []ghReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, fmt.Errorf("failed to decode github response: %w", err)
	}
	if len(versions) == 0 {
		return nil, ErrNoVersions
	}
	ver := versions[0]
	if ver.TagName == "" || !semver.IsValid(ver.TagName) {
		return nil, ErrNoVersions
	}
	return &ver, nil
}
