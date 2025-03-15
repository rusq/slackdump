package viewer

import (
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/fasttime"
	"github.com/rusq/slackdump/v3/internal/structures"
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
	Messages       iter.Seq2[slack.Message, error]
	ThreadMessages iter.Seq2[slack.Message, error]
	ThreadID       string
}

// view returns a mainView struct with the channels and the name and type of
// the source.
func (v *Viewer) view() mainView {
	return mainView{
		channels: v.ch,
		Name:     filepath.Base(v.src.Name()),
		Type:     v.src.Type().String(),
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

func maybeReverse(mm []slack.Message) error {
	if len(mm) == 0 {
		return nil
	}

	first, err := fasttime.TS2int(mm[0].Timestamp)
	if err != nil {
		return fmt.Errorf("TS2int at 0: %w", err)
	}
	last, err := fasttime.TS2int(mm[len(mm)-1].Timestamp)
	if err != nil {
		return fmt.Errorf("TS2int at -1: %w", err)
	}
	if first > last {
		slices.Reverse(mm)
	}
	return nil
}

func (v *Viewer) channelHandler(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	lg := v.lg.With("in", "channelHandler", "channel", id)
	it, err := v.src.AllMessages(r.Context(), id)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		lg.ErrorContext(ctx, "AllMessages", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lg.DebugContext(ctx, "conversation", "id", id)

	ci, err := v.src.ChannelInfo(r.Context(), id)
	if err != nil {
		lg.ErrorContext(ctx, "src.ChannelInfo", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := v.view()
	page.Conversation = *ci
	page.Messages = it

	template := "index.html" // for deep links
	if isHXRequest(r) {
		template = "hx_conversation"
	}
	if err := v.tmpl.ExecuteTemplate(w, template, page); err != nil {
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) postRedirectHandler(w http.ResponseWriter, r *http.Request, id string) {
	lg := v.lg.With("in", "postRedirectHandler", "channel", id)
	ts := r.PathValue("ts")
	if ts == "" || isInvalid(ts) {
		lg.Error("invalid ts", "ts", ts)
		http.Redirect(w, r, "/"+id, http.StatusSeeOther)
		return
	}
	if strings.HasPrefix(ts, "p") {
		values := r.URL.Query()
		if vts := values.Get("thread_ts"); vts != "" {
			// in this case the initial p value refers to a message within the thread
			// https://ora600.slack.com/archives/CHY5HUESG/p1738580940349469?thread_ts=1737716342.919259&cid=CHY5HUESG
			lg.Debug("redirecting to thread message", "ts", vts)
			http.Redirect(w, r, "/archives/"+id+"/"+vts+"#"+structures.ThreadIDtoTS(ts), http.StatusSeeOther)
		} else {
			// p refers to a message within the channel.
			// https: //ora600.slack.com/archives/DHMAB25DY/p1710063528879959
			lg.Debug("redirecting to channel message", "ts", ts)
			ts = structures.ThreadIDtoTS(ts)
			http.Redirect(w, r, "/archives/"+id+"#"+ts, http.StatusSeeOther)
		}
		return
	}
	lg.Debug("redirecting to thread message", "ts", ts)
	v.threadHandler(w, r, id)
}

func isHXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func isInvalid(path string) bool {
	return strings.Contains(path, "..") || strings.Contains(path, "~") || strings.Contains(path, "/") || strings.Contains(path, "\\")
}

func (v *Viewer) threadHandler(w http.ResponseWriter, r *http.Request, id string) {
	ts := r.PathValue("ts")
	if ts == "" || isInvalid(ts) {
		http.NotFound(w, r)
		return
	}

	if strings.HasPrefix(ts, "p") {
		ts = structures.ThreadIDtoTS(ts)
	}

	ctx := r.Context()
	lg := v.lg.With("in", "threadHandler", "channel", id, "thread", ts)
	itTm, err := v.src.AllThreadMessages(r.Context(), id, ts)
	if err != nil {
		lg.ErrorContext(ctx, "AllThreadMessages", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lg.DebugContext(ctx, "Messages")

	ci, err := v.src.ChannelInfo(r.Context(), id)
	if err != nil {
		lg.ErrorContext(ctx, "ChannelInfo", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := v.view()
	page.Conversation = *ci
	page.ThreadMessages = itTm
	page.ThreadID = ts

	var template string
	if isHXRequest(r) {
		template = "hx_thread"
	} else {
		template = "index.html"

		// if we're deep linking, channel view might not contain the messages,
		// so we need to fetch them.
		itMsg, err := v.src.AllMessages(r.Context(), id)
		if err != nil {
			lg.ErrorContext(ctx, "AllMessages", "error", err, "template", template)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		page.Messages = itMsg
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
	if id == "" || filename == "" || isInvalid(filename) || isInvalid(id) {
		http.NotFound(w, r)
		return
	}
	lg := v.lg.With("in", "fileHandler", "id", id, "filename", filename)
	fs := v.src.Files().FS()
	path, err := v.src.Files().File(id, filename)
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
