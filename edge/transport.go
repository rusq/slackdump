package edge

import "net/http"

// a simple wrapper for http.RoundTripper to do something before and after RoundTrip
type transport struct {
	tr        http.RoundTripper
	BeforeReq func(req *http.Request)
	AfterReq  func(resp *http.Response, req *http.Request)
}

func newTransport(tr http.RoundTripper) *transport {
	t := &transport{}
	if tr == nil {
		tr = http.DefaultTransport
	}
	t.tr = tr
	return t
}

func (t *transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	if t.BeforeReq != nil {
		t.BeforeReq(req)
	}
	resp, err = t.tr.RoundTrip(req)
	if err != nil {
		return
	}
	if t.AfterReq != nil {
		t.AfterReq(resp, req)
	}
	return
}
