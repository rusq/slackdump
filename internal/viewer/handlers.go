package viewer

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fasttime"
	"golang.org/x/exp/slices"
)

func (v *Viewer) indexHandler(w http.ResponseWriter, r *http.Request) {
	var page = struct {
		Conversation slack.Channel
		Name         string
		Type         string
		channels
	}{
		Conversation: slack.Channel{}, //blank.
		Name:         filepath.Base(v.src.Name()),
		Type:         v.src.Type(),
		channels:     v.ch,
	}
	if err := v.tmpl.ExecuteTemplate(w, "index.html", page); err != nil {
		v.lg.Printf("error: %v", err)
		//http.Error(w, err.Error(), http.StatusInternalServerError)
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
	var (
		id       = r.PathValue("id")
		filename = r.PathValue("filename")
	)
	if id == "" || filename == "" {
		http.NotFound(w, r)
		return
	}
	f, err := v.src.File(id, filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fi, err := f.Stat()
	if err != nil {
		v.lg.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, filename, fi.ModTime(), f.(io.ReadSeeker)) // TODO: hack
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
