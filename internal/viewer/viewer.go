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

	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rusq/slack"
	"golang.org/x/exp/slices"

	"github.com/rusq/slackdump/v3/internal/chunk"
	st "github.com/rusq/slackdump/v3/internal/structures"
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
	r   Renderer
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
		r:  &debugrender{},
	}
	// postinit
	{
		var tmpl = template.Must(template.New("").Funcs(
			template.FuncMap{
				"rendername":      v.name,
				"displayname":     v.um.DisplayName,
				"time":            localtime,
				"markdown":        v.r.RenderText,
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

func (v *Viewer) indexHandler(w http.ResponseWriter, r *http.Request) {
	var page = struct {
		Conversation slack.Channel
		Name         string
		channels
	}{
		Conversation: slack.Channel{}, //blank.
		Name:         v.d.Name(),
		channels:     v.ch,
	}
	if err := v.tmpl.ExecuteTemplate(w, "index.html", page); err != nil {
		v.lg.Printf("error: %v", err)
		//http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) newFileHandler(fn func(w http.ResponseWriter, r *http.Request, id string, f *chunk.File)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		f, err := v.d.Open(chunk.ToFileID(id, "", false))
		if err != nil {
			v.lg.Printf("%s: error: %v", id, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		fn(w, r, id, f)
	}
}

func (v *Viewer) channelHandler(w http.ResponseWriter, r *http.Request, id string, f *chunk.File) {
	mm, err := f.AllMessages(id)
	if err != nil {
		v.lg.Printf("%s: error: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(mm)

	v.lg.Debugf("conversation: %s, got %d messages", id, len(mm))

	ci, err := f.ChannelInfo(id)
	if err != nil {
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var page = struct {
		Conversation slack.Channel
		Messages     []slack.Message
	}{
		Conversation: *ci,
		Messages:     mm,
	}
	if err := v.tmpl.ExecuteTemplate(w, "hx_conversation", page); err != nil {
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

func (v *Viewer) threadHandler(w http.ResponseWriter, r *http.Request, id string, f *chunk.File) {
	ts := r.PathValue("ts")
	if ts == "" {
		http.NotFound(w, r)
		return
	}
	mm, err := f.AllThreadMessages(id, ts)
	if err != nil {
		v.lg.Printf("%s: error: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(mm)

	v.lg.Debugf("conversation: %s, thread: %s, got %d messages", id, ts, len(mm))

	ci, err := f.ChannelInfo(id)
	if err != nil {
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var page = struct {
		Conversation slack.Channel
		Messages     []slack.Message
		ThreadID     string
	}{
		ThreadID:     ts,
		Conversation: *ci,
		Messages:     mm,
	}
	if err := v.tmpl.ExecuteTemplate(w, "hx_thread", page); err != nil {
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) fileHandler(w http.ResponseWriter, r *http.Request) {
}

func (v *Viewer) userHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("user_id")
	if uid == "" {
		http.NotFound(w, r)
		return
	}
	u, found := v.um[uid]
	if !found {
		http.NotFound(w, r)
		return
	}
	spew.Dump(u)

	if err := v.tmpl.ExecuteTemplate(w, "hx_user", u); err != nil {
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func localtime(ts string) string {
	t, err := st.ParseSlackTS(ts)
	if err != nil {
		return ts
	}
	return t.Local().Format(time.DateTime)
}
