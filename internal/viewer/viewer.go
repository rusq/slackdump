// Package viewer implements the logic to view the slackdump files.
package viewer

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/rusq/slackdump/v3/internal/source"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rusq/slack"

	st "github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/internal/viewer/renderer"
)

var debug = os.Getenv("DEBUG") != ""

func init() {
	if debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
}

// Viewer is the slackdump viewer.
type Viewer struct {
	// data
	ch   channels
	um   st.UserIndex
	src  source.Sourcer
	tmpl *template.Template

	// handles
	srv *http.Server
	lg  *slog.Logger
	r   renderer.Renderer
}

const (
	hour = 60 * time.Minute
)

// New creates new viewer instance.  Once [Viewer.ListenAndServe] is called, the
// viewer will start serving the web interface on the given address.  The
// address should be in the form of ":8080". The viewer will use the given
// [Sourcer] to retrieve the data, see "source" package for available options.
// It will initialise the logger from the context.
func New(ctx context.Context, addr string, r source.Sourcer) (*Viewer, error) {
	all, err := r.Channels(ctx)
	if err != nil {
		return nil, err
	}
	cc := initChannels(all)

	uu, err := r.Users(ctx)
	if err != nil {
		return nil, err
	}
	um := st.NewUserIndex(uu)

	v := &Viewer{
		src: r,
		ch:  cc,
		um:  um,
		lg:  slog.Default(),
	}
	// postinit
	initTemplates(v)
	if debug {
		v.r = &renderer.Debug{}
	} else {
		opts := []renderer.SlackOption{
			renderer.WithUsers(indexusers(uu)),
			renderer.WithChannels(indexchannels(all)),
		}
		if wi, err := r.WorkspaceInfo(ctx); err == nil {
			opts = append(opts, renderer.WithReplaceURL(wi.URL, normalise(addr)))
		}
		v.r = renderer.NewSlack(
			v.tmpl,
			opts...,
		)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /", v.indexHandler)
	// https: //ora600.slack.com/archives/CHY5HUESG
	mux.HandleFunc("GET /archives/{id}", v.newFileHandler(v.channelHandler))
	// https: //ora600.slack.com/archives/DHMAB25DY/p1710063528879959
	// https://ora600.slack.com/archives/CHY5HUESG/p1738580940349469?thread_ts=1737716342.919259&cid=CHY5HUESG
	mux.HandleFunc("GET /archives/{id}/{ts}", v.newFileHandler(v.postRedirectHandler))
	mux.HandleFunc("GET /team/{user_id}", v.userHandler)
	mux.Handle("GET /slackdump/file/{id}/{filename}", cacheMwareFunc(3*hour)(http.HandlerFunc(v.fileHandler)))
	v.srv = &http.Server{
		Addr:    addr,
		Handler: middleware.Logger(mux),
	}

	return v, nil
}

func normalise(addr string) string {
	if addr == "" {
		return "127.0.0.1:8080"
	}
	if addr[0] == ':' {
		return "127.0.0.1" + addr
	}
	return addr
}

func (v *Viewer) ListenAndServe() error {
	return v.srv.ListenAndServe()
}

func (v *Viewer) Close() error {
	var ee error
	if err := v.srv.Close(); err != nil {
		ee = errors.Join(err)
	}
	v.lg.Debug("server closed")
	if ee != nil {
		v.lg.Error("close", "errors", ee)
	}
	return ee
}

func indexusers(uu []slack.User) map[string]slack.User {
	um := make(map[string]slack.User, len(uu))
	for _, u := range uu {
		um[u.ID] = u
	}
	return um
}

func indexchannels(cc []slack.Channel) map[string]slack.Channel {
	cm := make(map[string]slack.Channel, len(cc))
	for _, c := range cc {
		cm[c.ID] = c
	}
	return cm
}

type channels struct {
	Public  []slack.Channel
	Private []slack.Channel
	MPIM    []slack.Channel
	DM      []slack.Channel
}

func initChannels(c []slack.Channel) channels {
	var cc channels
	for _, ch := range c {
		t := st.ChannelType(ch)
		switch t {
		case st.CIM:
			cc.DM = append(cc.DM, ch)
		case st.CMPIM:
			cc.MPIM = append(cc.MPIM, ch)
		case st.CPrivate:
			cc.Private = append(cc.Private, ch)
		default:
			cc.Public = append(cc.Public, ch)
		}
	}
	return cc
}
