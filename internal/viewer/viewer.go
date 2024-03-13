// Package viewer implements the logic to view the slackdump files.
package viewer

import (
	"context"
	"embed"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	st "github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/internal/viewer/renderer"
	"github.com/rusq/slackdump/v3/logger"
)

//go:embed templates
var fsys embed.FS

type Viewer struct {
	// data
	ch   channels
	um   st.UserIndex
	d    *chunk.Directory
	tmpl *template.Template

	// handles
	srv *http.Server
	lg  logger.Interface
	r   renderer.Renderer
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
		t := st.ChannelType(c)
		switch t {
		case st.CIM:
			cc.DM = append(cc.DM, c)
		case st.CMPIM:
			cc.MPIM = append(cc.MPIM, c)
		case st.CPrivate:
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
		um: st.NewUserIndex(uu),
		lg: logger.FromContext(ctx),
		r:  &renderer.Slack{},
	}
	// postinit
	{
		var tmpl = template.Must(template.New("").Funcs(
			template.FuncMap{
				"rendername":      v.name,
				"displayname":     v.um.DisplayName,
				"time":            localtime,
				"rendertext":      v.r.RenderText, // render message text
				"render":          v.r.Render,     // render message
				"is_thread_start": st.IsThreadStart,
			},
		).ParseFS(fsys, "templates/*.html"))
		v.tmpl = tmpl
	}

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

func localtime(ts string) string {
	t, err := st.ParseSlackTS(ts)
	if err != nil {
		return ts
	}
	return t.Local().Format(time.DateTime)
}

func (v *Viewer) name(ch slack.Channel) (who string) {
	t := st.ChannelType(ch)
	switch t {
	case st.CIM:
		who = "@" + v.um.DisplayName(ch.User)
	case st.CMPIM:
		who = strings.Replace(ch.Purpose.Value, " messaging with", "", -1)
	case st.CPrivate:
		who = "🔒 " + ch.NameNormalized
	default:
		who = "#" + ch.NameNormalized
	}
	return who
}
