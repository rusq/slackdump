// Package edge provides a limited implementation of undocumented Slack Edge
// API necessary to get the data from a slack workspace.
package edge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/tagmagic"
)

type Client struct {
	cl           *http.Client
	edgeAPI      string
	webclientAPI string
	token        string

	teamID string
}

var (
	ErrNoTeamID = errors.New("teamID is empty")
	ErrNoToken  = errors.New("token is empty")
)

func NewWithClient(workspaceName string, teamID string, token string, cl *http.Client) (*Client, error) {
	if teamID == "" {
		return nil, fmt.Errorf("teamID is empty")
	}
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}
	return &Client{
		cl:           cl,
		token:        token,
		teamID:       teamID,
		webclientAPI: fmt.Sprintf("https://%s.slack.com/api/", workspaceName),
		edgeAPI:      fmt.Sprintf("https://edgeapi.slack.com/cache/%s/", teamID),
	}, nil
}

func NewWithToken(ctx context.Context, workspaceName string, teamID string, token string, cookies []*http.Cookie) (*Client, error) {
	prov, err := auth.NewValueCookiesAuth(token, cookies)
	if err != nil {
		return nil, err
	}
	return New(ctx, prov)
}

func New(ctx context.Context, prov auth.Provider) (*Client, error) {
	info, err := prov.Test(ctx)
	if err != nil {
		return nil, err
	}
	hcl, err := prov.HTTPClient()
	if err != nil {
		return nil, err
	}
	cl := &Client{
		cl:           hcl,
		token:        prov.SlackToken(),
		teamID:       info.TeamID,
		webclientAPI: info.URL + "api/",
		edgeAPI:      fmt.Sprintf("https://edgeapi.slack.com/cache/%s/", info.TeamID),
	}
	return cl, nil
}

func (cl *Client) Raw() *http.Client {
	return cl.cl
}

type BaseRequest struct {
	Token string `json:"token"`
}

type BaseResponse struct {
	Ok               bool             `json:"ok"`
	Error            string           `json:"error,omitempty"`
	ResponseMetadata ResponseMetadata `json:"response_metadata,omitempty"`
}

type ResponseMetadata struct {
	Messages   []string `json:"messages,omitempty"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

func (r *BaseRequest) SetToken(token string) {
	r.Token = token
}

func (r *BaseRequest) IsTokenSet() bool {
	return len(r.Token) > 0
}

type PostRequest interface {
	SetToken(string)
	IsTokenSet() bool
}

func (cl *Client) Post(ctx context.Context, path string, req PostRequest) (*http.Response, error) {
	if !req.IsTokenSet() {
		req.SetToken(cl.token)
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, cl.edgeAPI+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	return cl.cl.Do(r)
}

func (cl *Client) ParseResponse(req any, resp *http.Response) error {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	slog.Info("response", "body", string(data))
	dec := json.NewDecoder(bytes.NewReader(data))
	return dec.Decode(req)
}

func (cl *Client) PostForm(ctx context.Context, path string, form url.Values) (*http.Response, error) {
	return cl.PostFormRaw(ctx, cl.webclientAPI+path, form)
}

func (cl *Client) PostFormRaw(ctx context.Context, url string, form url.Values) (*http.Response, error) {
	if form["token"] == nil {
		form.Set("token", cl.token)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("accept-language", "en-NZ,en-AU;q=0.9,en;q=0.8,ru;q=0.7")
	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	return cl.cl.Do(req)
}

// values returns url.Values from a struct.  If omitempty is true, then the
// empty values are omitted for the fields that have the `omitempty` tag.
func values[T any](s T, omitempty bool) url.Values {
	var v = make(url.Values)
	m := tagmagic.ToMap(s, omitempty)
	for k, val := range m {
		v.Set(k, fmt.Sprint(val))
	}
	return v
}

func (cl *Client) webapiURL(endpoint string) string {
	return cl.webclientAPI + endpoint
}
