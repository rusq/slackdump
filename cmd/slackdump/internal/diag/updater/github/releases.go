// In this file: types for GitHub releases API responses used by the updater diagnostics.

package github

import (
	"time"
)

type Releases []Release

type Release struct {
	URL             string    `json:"url"`
	AssetsURL       string    `json:"assets_url"`
	UploadURL       string    `json:"upload_url"`
	HTMLURL         string    `json:"html_url"`
	ID              int64     `json:"id"`
	Author          Author    `json:"author"`
	NodeID          string    `json:"node_id"`
	TagName         string    `json:"tag_name"`
	TargetCommitish string    `json:"target_commitish"`
	Name            string    `json:"name"`
	Draft           bool      `json:"draft"`
	Immutable       bool      `json:"immutable"`
	Prerelease      bool      `json:"prerelease"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	PublishedAt     time.Time `json:"published_at"`
	Assets          []Asset   `json:"assets"`
	TarballURL      string    `json:"tarball_url"`
	ZipballURL      string    `json:"zipball_url"`
	Body            string    `json:"body"`
}

type Asset struct {
	URL                string    `json:"url"`
	ID                 int64     `json:"id"`
	NodeID             string    `json:"node_id"`
	Name               string    `json:"name"`
	Label              string    `json:"label"`
	Uploader           Author    `json:"uploader"`
	ContentType        string    `json:"content_type"`
	State              State     `json:"state"`
	Size               int64     `json:"size"`
	Digest             string    `json:"digest"`
	DownloadCount      int64     `json:"download_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	BrowserDownloadURL string    `json:"browser_download_url"`
}

type Author struct {
	Login             Login        `json:"login"`
	ID                int64        `json:"id"`
	NodeID            string       `json:"node_id"`
	AvatarURL         string       `json:"avatar_url"`
	GravatarID        string       `json:"gravatar_id"`
	URL               string       `json:"url"`
	HTMLURL           string       `json:"html_url"`
	FollowersURL      string       `json:"followers_url"`
	FollowingURL      string       `json:"following_url"`
	GistsURL          string       `json:"gists_url"`
	StarredURL        string       `json:"starred_url"`
	SubscriptionsURL  string       `json:"subscriptions_url"`
	OrganizationsURL  string       `json:"organizations_url"`
	ReposURL          string       `json:"repos_url"`
	EventsURL         string       `json:"events_url"`
	ReceivedEventsURL string       `json:"received_events_url"`
	Type              Type         `json:"type"`
	UserViewType      UserViewType `json:"user_view_type"`
	SiteAdmin         bool         `json:"site_admin"`
}

type State string

const (
	Uploaded State = "uploaded"
)

type Login string

const (
	GithubActionsBot Login = "github-actions[bot]"
)

type Type string

const (
	Bot Type = "Bot"
)

type UserViewType string

const (
	Public UserViewType = "public"
)
