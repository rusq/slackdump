// Package viewer implements the logic to view the slackdump files.
package viewer

import (
	"context"
	"embed"
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/logger"
)

//go:embed templates
var fsys embed.FS

type Viewer struct {
	ch   channels
	um   structures.UserIndex
	d    *chunk.Directory
	srv  *http.Server
	lg   logger.Interface
	tmpl *template.Template
}

type channels struct {
	Public  []slack.Channel
	Private []slack.Channel
	MPIM    []slack.Channel
	DM      []slack.Channel
}

func New(ctx context.Context, dir *chunk.Directory, addr string) (*Viewer, error) {
	all, err := dir.Channels()
	if err != nil {
		return nil, err
	}
	var cc channels
	for _, c := range all {
		t := channelType(c)
		switch t {
		case CIM:
			cc.DM = append(cc.DM, c)
		case CMPIM:
			cc.MPIM = append(cc.MPIM, c)
		case CPrivate:
			cc.Private = append(cc.Private, c)
		default:
			cc.Public = append(cc.Public, c)
		}
	}
	uu, err := dir.Users()
	if err != nil {
		return nil, err
	}

	v := &Viewer{
		d:  dir,
		ch: cc,
		um: structures.NewUserIndex(uu),
		lg: logger.FromContext(ctx),
	}
	// postinit
	{
		var tmpl = template.Must(template.New("").Funcs(
			template.FuncMap{
				"rendername": v.name,
			},
		).ParseFS(fsys, "templates/*.html"))
		v.tmpl = tmpl
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", v.indexHandler)
	// https: //ora600.slack.com/archives/CHY5HUESG
	mux.HandleFunc("/archives/{id}", v.channelHandler)
	// https: //ora600.slack.com/archives/DHMAB25DY/p1710063528879959
	mux.HandleFunc("/channel/{id}/{ts}", v.threadHandler)
	mux.HandleFunc("/files/{id}", v.fileHandler)
	v.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return v, nil
}

func (v *Viewer) ListenAndServe() error {
	return v.srv.ListenAndServe()
}

func (v *Viewer) Close() error {
	var ee error
	if err := v.d.Close(); err != nil {
		ee = errors.Join(err)
	}
	if err := v.srv.Close(); err != nil {
		ee = errors.Join(err)
	}
	v.lg.Debug("server closed")
	if ee != nil {
		v.lg.Printf("errors: %v", ee)
	}
	return ee
}

func (v *Viewer) indexHandler(w http.ResponseWriter, r *http.Request) {
	if err := v.tmpl.ExecuteTemplate(w, "index.html", v.ch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) channelHandler(w http.ResponseWriter, r *http.Request) {
}

const (
	CUnknown = iota
	CIM
	CMPIM
	CPrivate
	CPublic
)

func channelType(ch slack.Channel) int {
	switch {
	case ch.IsIM:
		return CIM
	case ch.IsMpIM:
		return CMPIM
	case ch.IsPrivate:
		return CPrivate
	default:
		return CPublic
	}
}

func (v *Viewer) name(ch slack.Channel) (who string) {
	t := channelType(ch)
	switch t {
	case CIM:
		who = "@" + v.um.DisplayName(ch.User)
	case CMPIM:
		who = strings.Replace(ch.Purpose.Value, " messaging with", "", -1)
	case CPrivate:
		who = "🔒 " + ch.NameNormalized
	default:
		who = "#" + ch.NameNormalized
	}
	return who
}

func (v *Viewer) threadHandler(w http.ResponseWriter, r *http.Request) {
}

func (v *Viewer) fileHandler(w http.ResponseWriter, r *http.Request) {
}
