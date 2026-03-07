// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package viewer

import (
	"errors"
	"io"
	"io/fs"
	"iter"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/rusq/slackdump/v4/source"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/structures"
)

func (v *Viewer) indexHandler(w http.ResponseWriter, r *http.Request) {
	page := v.view()
	if err := v.tmpl.ExecuteTemplate(w, "index.html", page); err != nil {
		v.lg.ErrorContext(r.Context(), "indexHandler", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// messageView is the data passed to the render_message template.
// It wraps a slack.Message with the channel ID needed to render threading
// context (reply-to anchor links).  Set ChannelID to an empty string to
// suppress the reply banner (used in the thread panel, where the parent
// message is on a different page).
type messageView struct {
	Msg       slack.Message
	ChannelID string
}

type mainView struct {
	channels
	Name            string
	Type            string
	Messages        iter.Seq2[slack.Message, error]
	ThreadMessages  iter.Seq2[slack.Message, error]
	ThreadID        string
	Conversation    slack.Channel
	Alias           string // conversation alias
	AliasError      string
	CanAlias        bool // if true, alias can be set for the channel
	CanvasActive    bool // true when the canvas tab is the active tab
	CanvasAvailable bool // true when the canvas file exists in storage
}

type Aliaser interface {
	// Alias returns the alias for the given channel ID.
	Alias(id string) (string, bool, error)
	// SetAlias sets the alias for the given channel ID.
	SetAlias(id, alias string) error
	// DeleteAlias deletes the alias for the given channel ID.
	DeleteAlias(id string) error
	// Aliases returns the list of aliases.
	Aliases() (map[string]string, error)
}

// view returns a mainView struct with the channels and the name and type of
// the source.
func (v *Viewer) view() mainView {
	_, supportsAlias := v.aliaser()

	return mainView{
		channels: v.ch,
		Name:     filepath.Base(v.src.Name()),
		Type:     v.src.Type().String(),
		CanAlias: supportsAlias,
	}
}

type aliasAction int

const (
	aliasDelete aliasAction = iota
	aliasSet
)

func (v *Viewer) aliaser() (Aliaser, bool) {
	a, ok := v.src.(Aliaser)
	return a, ok
}

func (v *Viewer) alias(id string) (string, bool, error) {
	a, ok := v.aliaser()
	if !ok {
		return "", false, nil
	}
	return a.Alias(id)
}

func validateAlias(raw string) (string, aliasAction, error) {
	alias := strings.TrimSpace(raw)
	if alias == "" {
		return "", aliasDelete, nil
	}
	if len([]rune(alias)) > 30 {
		return "", aliasSet, errors.New("alias must be 30 characters or fewer")
	}
	for _, r := range alias {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			continue
		}
		return "", aliasSet, errors.New("alias may only contain letters, digits, underscores, and dashes")
	}
	return alias, aliasSet, nil
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
	it, err := v.src.AllMessages(r.Context(), id)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
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
	if err := v.setConversation(&page, ci); err != nil {
		lg.ErrorContext(ctx, "setConversation", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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

// isInvalid returns true if the provided path component is not web-safe.
func isInvalid(pcomp string) bool {
	return strings.Contains(pcomp, "..") || strings.HasPrefix(pcomp, "~") || strings.Contains(pcomp, "/") || strings.Contains(pcomp, "\\")
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
	if err := v.setConversation(&page, ci); err != nil {
		lg.ErrorContext(ctx, "setConversation", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
	fsys := v.src.Files().FS()
	path, err := v.src.Files().File(id, filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		lg.ErrorContext(ctx, "File", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Setting content type to application/octet-stream to support any files without extensions (e.g. Canvas files)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFileFS(w, r, fsys, path)
}

func (v *Viewer) userHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("user_id")
	if uid == "" {
		http.NotFound(w, r)
		return
	}
	lg := v.lg.With("in", "userHandler", "user_id", uid)
	u, found := v.um[uid]
	if !found {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()
	if err := v.tmpl.ExecuteTemplate(w, "hx_user", u); err != nil {
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) aliasHandler(w http.ResponseWriter, r *http.Request) {
	if !v.canAlias() {
		http.NotFound(w, r)
		return
	}
	chanID := r.PathValue("id")
	if chanID == "" {
		http.NotFound(w, r)
		return
	}
	ctx := r.Context()
	lg := v.lg.With("in", "aliasHandler", "channel_id", chanID)
	ch, found := v.ch.find(chanID)
	if !found {
		http.NotFound(w, r)
		return
	}
	slog.DebugContext(ctx, "aliasHandler", "channel", ch.Name, "id", ch.ID)
	view := v.view()
	view.Conversation = ch
	if alias, ok, err := v.alias(chanID); err != nil {
		lg.ErrorContext(ctx, "Alias", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if ok {
		view.Alias = alias
	}

	if err := v.tmpl.ExecuteTemplate(w, "hx_alias", view); err != nil {
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// canvasAvailable returns true if the canvas file for the channel exists in
// the given storage.
func canvasAvailable(storage source.Storage, ci *slack.Channel) bool {
	if ci.Properties == nil || ci.Properties.Canvas.FileId == "" {
		return false
	}
	_, err := fileByID(storage, ci.Properties.Canvas.FileId)
	return err == nil
}

// setConversation sets the Conversation and canvas-related fields on the page
// view.  It should be called whenever a channel is loaded, so that the canvas
// tab state is always consistent regardless of which tab is active.
func (v *Viewer) setConversation(page *mainView, ci *slack.Channel) error {
	page.Conversation = *ci
	page.CanvasAvailable = canvasAvailable(v.src.Files(), ci)
	if alias, ok, err := v.alias(ci.ID); err != nil {
		return err
	} else if ok {
		page.Alias = alias
	}
	return nil
}

// canvasHandler renders the canvas tab view for the given channel.
func (v *Viewer) canvasHandler(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	lg := v.lg.With("in", "canvasHandler", "channel", id)

	ci, err := v.src.ChannelInfo(ctx, id)
	if err != nil {
		lg.ErrorContext(ctx, "ChannelInfo", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := v.view()
	if err := v.setConversation(&page, ci); err != nil {
		lg.ErrorContext(ctx, "setConversation", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	page.CanvasActive = true

	tmplName := "hx_canvas"
	if !isHXRequest(r) {
		tmplName = "index.html"
		// fetch messages so the full page renders correctly on deep link
		itMsg, err := v.src.AllMessages(ctx, id)
		if err != nil {
			lg.ErrorContext(ctx, "AllMessages", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		page.Messages = itMsg
	}
	if err := v.tmpl.ExecuteTemplate(w, tmplName, page); err != nil {
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err, "template", tmplName)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) canAlias() bool {
	_, ok := v.aliaser()
	return ok
}

func (v *Viewer) aliasPutHandler(w http.ResponseWriter, r *http.Request) {
	if !v.canAlias() {
		http.NotFound(w, r)
		return
	}

	lg := v.lg.With("in", "aliasPutHandler")
	lg.Debug("aliasPutHandler")

	chanID := r.PathValue("id")
	if chanID == "" {
		http.NotFound(w, r)
		return
	}
	lg = lg.With("channel_id", chanID)
	ctx := r.Context()
	ch, found := v.ch.find(chanID)
	if !found {
		http.NotFound(w, r)
		return
	}
	slog.DebugContext(ctx, "aliasPutHandler", "channel", ch.Name, "id", ch.ID)

	view := v.view()
	view.Conversation = ch
	alias, action, err := validateAlias(r.FormValue("alias"))
	if err != nil {
		view.Alias = strings.TrimSpace(r.FormValue("alias"))
		view.AliasError = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		if execErr := v.tmpl.ExecuteTemplate(w, "hx_alias", view); execErr != nil {
			lg.ErrorContext(ctx, "ExecuteTemplate", "error", execErr)
			http.Error(w, execErr.Error(), http.StatusInternalServerError)
		}
		return
	}
	aliaser, _ := v.aliaser()
	switch action {
	case aliasDelete:
		err = aliaser.DeleteAlias(chanID)
	default:
		err = aliaser.SetAlias(chanID, alias)
	}
	if err != nil {
		lg.ErrorContext(ctx, "persist alias", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := v.setConversation(&view, &ch); err != nil {
		lg.ErrorContext(ctx, "setConversation", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := v.tmpl.ExecuteTemplate(w, "hx_alias_response", view); err != nil {
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) aliasDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !v.canAlias() {
		http.NotFound(w, r)
		return
	}
	lg := v.lg.With("in", "aliasDeleteHandler")
	lg.Debug("aliasDeleteHandler")
	chanID := r.PathValue("id")
	if chanID == "" {
		http.NotFound(w, r)
		return
	}
	lg = lg.With("channel_id", chanID)
	ctx := r.Context()
	ch, found := v.ch.find(chanID)
	if !found {
		lg.Debug("not found")
		http.NotFound(w, r)
		return
	}
	slog.DebugContext(ctx, "aliasDeleteHandler", "channel", ch.Name, "id", ch.ID)
	aliaser, _ := v.aliaser()
	if err := aliaser.DeleteAlias(chanID); err != nil {
		lg.ErrorContext(ctx, "DeleteAlias", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	view := v.view()
	if err := v.setConversation(&view, &ch); err != nil {
		lg.ErrorContext(ctx, "setConversation", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := v.tmpl.ExecuteTemplate(w, "hx_alias_response", view); err != nil {
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// canvasContentHandler streams the raw canvas HTML content for the given
// channel directly, without requiring the caller to know the filename.
func (v *Viewer) canvasContentHandler(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	lg := v.lg.With("in", "canvasContentHandler", "channel", id)

	ci, err := v.src.ChannelInfo(ctx, id)
	if err != nil {
		lg.ErrorContext(ctx, "ChannelInfo", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if ci.Properties == nil || ci.Properties.Canvas.FileId == "" {
		http.NotFound(w, r)
		return
	}

	fileID := ci.Properties.Canvas.FileId
	storage := v.src.Files()
	pth, err := fileByID(storage, fileID)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			lg.DebugContext(ctx, "canvas file not found", "fileID", fileID, "error", err)
			http.NotFound(w, r)
			return
		}
		lg.ErrorContext(ctx, "FileByID", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f, err := storage.FS().Open(pth)
	if err != nil {
		lg.ErrorContext(ctx, "Open", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := io.Copy(w, f); err != nil {
		lg.ErrorContext(ctx, "Copy", "error", err)
	}
}
