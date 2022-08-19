// Package edge provides a limited implementation of undocumented Slack Edge
// API necessary to get the data from a slack workspace.
package edge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/rusq/slackdump/v2/auth"
	"golang.org/x/net/publicsuffix"
)

type Client struct {
	cl      *http.Client
	apiPath string
	token   string
}

func New(teamID string, token string, cookies []*http.Cookie) *Client {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	url, err := url.Parse("https://slack.com")
	if err != nil {
		panic(err) //shouldn't happen
	}
	jar.SetCookies(url, cookies)
	tr := newTransport(nil)
	tr.BeforeReq = func(req *http.Request) {
		// req.V
		// if req.Method == http.MethodGet {
		// 	req.Form.Add("token", token)
		// }
	}
	cl := http.Client{
		Jar:       jar,
		Transport: tr,
	}
	return &Client{
		cl:      &cl,
		token:   token,
		apiPath: fmt.Sprintf("https://edgeapi.slack.com/cache/%s/", teamID)}
}

func NewWithProvider(teamID string, prov auth.Provider) *Client {
	return New(teamID, prov.SlackToken(), sliceOfPtr(prov.Cookies()))
}

func (cl *Client) Raw() *http.Client {
	return cl.cl
}

type BaseRequest struct {
	Token string `json:"token"`
}

type BaseResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
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

func (cl *Client) Post(_ context.Context, path string, req PostRequest) (*http.Response, error) {
	if !req.IsTokenSet() {
		req.SetToken(cl.token)
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return cl.cl.Post(cl.apiPath+path, "application/json", bytes.NewReader(data))
}

func (cl *Client) ParseResponse(req any, resp *http.Response) error {
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(req)
}

func sliceOfPtr[T any](cc []T) []*T {
	var ret = make([]*T, len(cc))
	for i := range cc {
		ret[i] = &cc[i]
	}
	return ret
}
