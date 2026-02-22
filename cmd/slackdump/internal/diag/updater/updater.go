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

// Package updater exposes update and auto-update functions.
// It uses the following terminology:
//   - "remote" - the latest version on the server, and
//   - "local" - a version of slackdump currently installed.
package updater

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/diag/updater/github"
)

type Updater struct {
	cl fetcher
}

//go:generate mockgen -source=updater.go -destination=updater_mock_test.go -package=updater
type fetcher interface {
	Latest(ctx context.Context) (*github.Release, error)
	ByTag(ctx context.Context, tag string) (*github.Release, error)
}

type Release struct {
	Version     string
	PublishedAt time.Time
}

// Equal returns true if the two releases have the same version and publish date.
// Version comparison is case-insensitive.
func (r Release) Equal(other Release) bool {
	return strings.EqualFold(r.Version, other.Version) && r.PublishedAt.Equal(other.PublishedAt)
}

func NewUpdater() Updater {
	return Updater{
		cl: &github.Client{
			Owner:      "rusq",
			Repo:       "slackdump",
			Prerelease: false,
		},
	}
}

func (u Updater) Latest(ctx context.Context) (Release, error) {
	var r Release
	latest, err := u.cl.Latest(ctx)
	if err != nil {
		return r, fmt.Errorf("error fetching the latest release information: %w", err)
	}
	r.Version = latest.TagName
	r.PublishedAt = latest.PublishedAt
	return r, nil
}

var ErrUnreleased = errors.New("current version is not released")

func (u Updater) Current(ctx context.Context) (Release, error) {
	var r Release

	if !cfg.Version.IsReleased() {
		return r, ErrUnreleased
	}

	rel, err := u.cl.ByTag(ctx, cfg.Version.Version)
	if err != nil {
		return r, err
	}
	r.Version = rel.TagName
	r.PublishedAt = rel.PublishedAt
	return r, nil
}
