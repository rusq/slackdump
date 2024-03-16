// Package viewer implements the logic to view the slackdump files.
package viewer

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rusq/slack"

	st "github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/internal/viewer/renderer"
	"github.com/rusq/slackdump/v3/logger"
)

type Viewer struct {
	// data
	ch   channels
	um   st.UserIndex
	rtr  Sourcer
	tmpl *template.Template

	// handles
	srv *http.Server
	lg  logger.Interface
	r   renderer.Renderer
}

// Sourcer is an interface for retrieving data from different sources.
type Sourcer interface {
	// Name should return the name of the retriever underlying media, i.e.
	// directory or archive.
	Name() string
	// Channels should return all channels.
	Channels() ([]slack.Channel, error)
	// Users should return all users.
	Users() ([]slack.User, error)
	// AllMessages should return all messages for the given channel id.
	AllMessages(channelID string) ([]slack.Message, error)
	// AllThreadMessages should return all messages for the given tuple
	// (channelID, threadID).
	AllThreadMessages(channelID, threadID string) ([]slack.Message, error)
	// ChannelInfo should return the channel information for the given channel
	// id.
	ChannelInfo(channelID string) (*slack.Channel, error)
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

func New(ctx context.Context, addr string, r Sourcer) (*Viewer, error) {
	all, err := r.Channels()
	if err != nil {
		return nil, err
	}
	cc := initChannels(all)

	uu, err := r.Users()
	if err != nil {
		return nil, err
	}
	um := st.NewUserIndex(uu)

	sr := renderer.NewSlack(renderer.WithUsers(indexusers(uu)), renderer.WithChannels(indexchannels(all)))
	// sr := &renderer.Debug{}
	v := &Viewer{
		rtr: r,
		ch:  cc,
		um:  um,
		lg:  logger.FromContext(ctx),
		r:   sr,
	}
	// postinit
	initTemplates(v)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", v.indexHandler)
	// https: //ora600.slack.com/archives/CHY5HUESG
	mux.HandleFunc("/archives/{id}", v.newFileHandler(v.channelHandler))
	// https: //ora600.slack.com/archives/DHMAB25DY/p1710063528879959
	mux.HandleFunc("/archives/{id}/{ts}", v.newFileHandler(v.threadHandler))
	mux.HandleFunc("/files/{id}", v.fileHandler)
	mux.HandleFunc("/team/{user_id}", v.userHandler)
	v.srv = &http.Server{
		Addr:    addr,
		Handler: middleware.Logger(mux),
	}

	return v, nil
}

func init() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
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
		v.lg.Printf("errors: %v", ee)
	}
	return ee
}

func indexusers(uu []slack.User) (m map[string]slack.User) {
	m = make(map[string]slack.User, len(uu))
	for i := range uu {
		m[uu[i].ID] = uu[i]
	}
	return m
}

func indexchannels(cc []slack.Channel) (m map[string]slack.Channel) {
	m = make(map[string]slack.Channel, len(cc))
	for i := range cc {
		m[cc[i].ID] = cc[i]
	}
	return m
}
