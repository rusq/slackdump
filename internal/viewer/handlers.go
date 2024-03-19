package viewer

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fasttime"
	"golang.org/x/exp/slices"
)

func (v *Viewer) indexHandler(w http.ResponseWriter, r *http.Request) {
	page := v.view()
	if err := v.tmpl.ExecuteTemplate(w, "index.html", page); err != nil {
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type mainView struct {
	channels
	Name           string
	Type           string
	Conversation   slack.Channel
	Messages       []slack.Message
	ThreadMessages []slack.Message
	ThreadID       string
}

// view returns a mainView struct with the channels and the name and type of
// the source.
func (v *Viewer) view() mainView {
	return mainView{
		channels: v.ch,
		Name:     filepath.Base(v.src.Name()),
		Type:     v.src.Type(),
	}
}

func (v *Viewer) newFileHandler(fn func(w http.ResponseWriter, r *http.Request, id string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		fn(w, r, id)
	}
}

func (v *Viewer) channelHandler(w http.ResponseWriter, r *http.Request, id string) {
	mm, err := v.src.AllMessages(id)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		v.lg.Printf("%s: error: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(mm) > 0 {
		first, err := fasttime.TS2int(mm[0].Timestamp)
		if err != nil {
			v.lg.Printf("error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		last, err := fasttime.TS2int(mm[len(mm)-1].Timestamp)
		if err != nil {
			v.lg.Printf("error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if first > last {
			slices.Reverse(mm)
		}
	}

	v.lg.Debugf("conversation: %s, got %d messages", id, len(mm))

	ci, err := v.src.ChannelInfo(id)
	if err != nil {
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := v.view()
	page.Conversation = *ci
	page.Messages = mm

	template := "index.html" // for deep links
	if isHXRequest(r) {
		template = "hx_conversation"
	}
	if err := v.tmpl.ExecuteTemplate(w, template, page); err != nil {
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func isHXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func (v *Viewer) threadHandler(w http.ResponseWriter, r *http.Request, id string) {
	ts := r.PathValue("ts")
	if ts == "" {
		http.NotFound(w, r)
		return
	}
	mm, err := v.src.AllThreadMessages(id, ts)
	if err != nil {
		v.lg.Printf("%s: error: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	v.lg.Debugf("conversation: %s, thread: %s, got %d messages", id, ts, len(mm))

	ci, err := v.src.ChannelInfo(id)
	if err != nil {
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := v.view()
	page.Conversation = *ci
	page.ThreadMessages = mm
	page.ThreadID = ts

	var template string
	if isHXRequest(r) {
		template = "hx_thread"
	} else {
		template = "index.html"

		// if we're deep linking, channel view might not contain the messages,
		// so we need to fetch them.
		msg, err := v.src.AllMessages(id)
		if err != nil {
			v.lg.Printf("error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		page.Messages = msg
	}
	if err := v.tmpl.ExecuteTemplate(w, template, page); err != nil {
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) fileHandler(w http.ResponseWriter, r *http.Request) {
	var (
		id       = r.PathValue("id")
		filename = r.PathValue("filename")
	)
	if id == "" || filename == "" {
		http.NotFound(w, r)
		return
	}
	fs := v.src.FS()
	path, err := v.src.File(id, filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.ServeFileFS(w, r, fs, path)
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
