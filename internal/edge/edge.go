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
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/tagmagic"
	"golang.org/x/time/rate"
)

type Client struct {
	// cl is the http client to use
	cl *http.Client
	// edgeAPI is the edge API endpoint
	edgeAPI string
	// webclientAPI is the webclient APIs endpoint
	webclientAPI string
	// token is the slack token
	token string

	// teamID is the team ID
	teamID string
}

type tier struct {
	// once every
	t time.Duration
	// burst
	b int
}

func (t tier) limiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(t.t), t.b)
}

var (
	// Tier1 is the first tier
	tier1 = tier{t: 1 * time.Minute, b: 2}
	tier2 = tier{t: 3 * time.Second, b: 3}
	tier3 = tier{t: 1200 * time.Millisecond, b: 4}
	tier4 = tier{t: 60 * time.Millisecond, b: 5}
)

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

func (r BaseResponse) validate(ep string) error {
	if !r.Ok {
		return &APIError{Err: r.Error, Metadata: r.ResponseMetadata, Endpoint: ep}
	}
	return nil
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

func (cl *Client) EdgePost(ctx context.Context, path string, req PostRequest) (*http.Response, error) {
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
	if resp.StatusCode < http.StatusOK || http.StatusMultipleChoices <= resp.StatusCode {
		return fmt.Errorf("error:  status code: %s", resp.Status)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(req); err != nil {
		return err
	}
	return nil
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
	req.Header.Set("Accept-Language", "en-NZ,en-AU;q=0.9,en;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	resp, err := cl.cl.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		strWait := resp.Header.Get("Retry-After")
		if strWait == "" {
			return nil, errors.New("got rate limited, but did not get a Retry-After header")
		}
		wait, err := time.ParseDuration(strWait + "s")
		if err != nil {
			return nil, err
		}
		time.Sleep(wait)
		resp, err = cl.cl.Do(req)
		if err != nil {
			return nil, err
		}
		// if we are still rate limited, then we are in trouble
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, errors.New("still rate limited after waiting")
		}
	}
	if resp.StatusCode < http.StatusOK || http.StatusMultipleChoices <= resp.StatusCode {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("error:  status code: %s, body: %s", resp.Status, string(body))
	}
	return resp, err
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

type APIError struct {
	Err      string
	Metadata ResponseMetadata
	Endpoint string
}

func (e *APIError) Error() string {
	if len(e.Metadata.Messages) > 0 {
		return e.Err + ": " + e.Metadata.Messages[0]
	}
	return e.Err
}

type WebClientFields struct {
	XReason  string `json:"_x_reason"`
	XMode    string `json:"_x_mode"`
	XSonic   bool   `json:"_x_sonic"`
	XAppName string `json:"_x_app_name"`
}

func webclientReason(reason string) WebClientFields {
	return WebClientFields{
		XReason:  reason,
		XMode:    "online",
		XSonic:   true,
		XAppName: "client",
	}
}
