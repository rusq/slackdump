package viewer

import (
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"golang.org/x/exp/slices"
)

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
