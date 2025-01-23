package viewer

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/davecgh/go-spew/spew"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/fasttime"
)

func (v *Viewer) indexHandler(w http.ResponseWriter, r *http.Request) {
	page := v.view()
	if err := v.tmpl.ExecuteTemplate(w, "index.html", page); err != nil {
		v.lg.ErrorContext(r.Context(), "indexHandler", "error", err)
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
	ctx := r.Context()
	lg := v.lg.With("in", "channelHandler", "channel", id)
	mm, err := v.src.AllMessages(id)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		lg.ErrorContext(ctx, "AllMessages", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(mm) > 0 {
		first, err := fasttime.TS2int(mm[0].Timestamp)
		if err != nil {
			lg.ErrorContext(ctx, "TS2int", "idx", 0, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		last, err := fasttime.TS2int(mm[len(mm)-1].Timestamp)
		if err != nil {
			lg.ErrorContext(ctx, "TS2int", "idx", -1, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if first > last {
			slices.Reverse(mm)
		}
	}

	lg.DebugContext(ctx, "conversation", "id", id, "message_count", len(mm))

	ci, err := v.src.ChannelInfo(r.Context(), id)
	if err != nil {
		lg.ErrorContext(ctx, "src.ChannelInfo", "error", err)
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
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err)
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
	ctx := r.Context()
	lg := v.lg.With("in", "threadHandler", "channel", id, "thread", ts)
	mm, err := v.src.AllThreadMessages(id, ts)
	if err != nil {
		lg.ErrorContext(ctx, "AllThreadMessages", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lg.DebugContext(ctx, "Messages", "mm_count", len(mm))

	ci, err := v.src.ChannelInfo(r.Context(), id)
	if err != nil {
		lg.ErrorContext(ctx, "ChannelInfo", "error", err)
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
			lg.ErrorContext(ctx, "AllMessages", "error", err, "template", template)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		page.Messages = msg
	}
	if err := v.tmpl.ExecuteTemplate(w, template, page); err != nil {
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err, "template", template)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) fileHandler(w http.ResponseWriter, r *http.Request) {
	var (
		id       = r.PathValue("id")
		filename = r.PathValue("filename")
		ctx      = r.Context()
	)
	if id == "" || filename == "" {
		http.NotFound(w, r)
		return
	}
	lg := v.lg.With("in", "fileHandler", "id", id, "filename", filename)
	fs := v.src.FS()
	path, err := v.src.File(id, filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		lg.ErrorContext(ctx, "File", "error", err)
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
	ctx := r.Context()
	lg := v.lg.With("in", "userHandler", "user_id", uid)
	u, found := v.um[uid]
	if !found {
		http.NotFound(w, r)
		return
	}
	spew.Dump(u)

	if err := v.tmpl.ExecuteTemplate(w, "hx_user", u); err != nil {
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
