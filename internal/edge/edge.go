// Package edge provides a limited implementation of undocumented Slack Edge
// API necessary to get the data from a slack workspace.
package edge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rusq/chttp"
	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/tagmagic"
)

type Client struct {
	cl      *http.Client
	apiPath string
	token   string

	teamID        string
	workspaceName string
}

func NewWithClient(workspaceName string, teamID string, token string, cl *http.Client) (*Client, error) {
	if teamID == "" {
		return nil, fmt.Errorf("teamID is empty")
	}
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}
	return &Client{
		workspaceName: workspaceName,
		cl:            cl,
		token:         token,
		apiPath:       fmt.Sprintf("https://edgeapi.slack.com/cache/%s/", teamID),
	}, nil
}

func New(workspaceName string, teamID string, token string, cookies []*http.Cookie) (*Client, error) {
	cl, err := chttp.New(auth.SlackURL, cookies)
	if err != nil {
		return nil, err
	}
	return NewWithClient(workspaceName, teamID, token, cl)
}

func NewWithProvider(workspaceName string, teamID string, prov auth.Provider) (*Client, error) {
	hcl, err := prov.HTTPClient()
	if err != nil {
		return nil, err
	}
	return NewWithClient(workspaceName, teamID, prov.SlackToken(), hcl)
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
	Messages []string `json:"messages,omitempty"`
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
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, cl.apiPath+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	return cl.cl.Do(r)
}

func (cl *Client) ParseResponse(req any, resp *http.Response) error {
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(req)
}

func (cl *Client) PostForm(ctx context.Context, path string, form url.Values) (*http.Response, error) {
	return cl.PostFormRaw(ctx, cl.apiPath+path, form)
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
