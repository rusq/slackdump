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
	"context"
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

// ── Render* methods ──────────────────────────────────────────────────────────
// These implement PageRenderer and never reference *http.Request.
// They always produce a complete HTML page (index.html).

// RenderIndex renders the channel-list index page to w.
func (v *Viewer) RenderIndex(ctx context.Context, w io.Writer) error {
	page := v.view()
	return v.tmpl.ExecuteTemplate(w, "index.html", page)
}

// RenderChannel renders the full conversation page for channelID to w.
func (v *Viewer) RenderChannel(ctx context.Context, channelID string, w io.Writer) error {
	ci, err := v.src.ChannelInfo(ctx, channelID)
	if err != nil {
		return err
	}
	it, err := v.allMessagesOrEmpty(ctx, channelID)
	if err != nil {
		return err
	}
	page := v.view()
	if err := v.setConversation(&page, ci); err != nil {
		return err
	}
	page.Messages = it
	return v.tmpl.ExecuteTemplate(w, "index.html", page)
}

// RenderThread renders the full thread page for (channelID, threadTS) to w.
// threadTS must be a clean timestamp (not the "p…" Slack URL form).
func (v *Viewer) RenderThread(ctx context.Context, channelID, threadTS string, w io.Writer) error {
	itTm, err := v.src.AllThreadMessages(ctx, channelID, threadTS)
	if err != nil {
		return err
	}
	ci, err := v.src.ChannelInfo(ctx, channelID)
	if err != nil {
		return err
	}
	page := v.view()
	if err := v.setConversation(&page, ci); err != nil {
		return err
	}
	page.ThreadMessages = itTm
	page.ThreadID = threadTS

	// fetch channel messages so the full page renders correctly on deep link.
	itMsg, err := v.src.AllMessages(ctx, channelID)
	if err != nil {
		return err
	}
	page.Messages = itMsg

	return v.tmpl.ExecuteTemplate(w, "index.html", page)
}

// RenderUser renders the full user-profile page for userID to w.
func (v *Viewer) RenderUser(ctx context.Context, userID string, w io.Writer) error {
	u, found := v.um[userID]
	if !found {
		return fs.ErrNotExist
	}
	page := v.view()
	page.User = u
	return v.tmpl.ExecuteTemplate(w, "index.html", page)
}

// RenderCanvas renders the full canvas tab page for channelID to w.
func (v *Viewer) RenderCanvas(ctx context.Context, channelID string, w io.Writer) error {
	ci, err := v.src.ChannelInfo(ctx, channelID)
	if err != nil {
		return err
	}
	page := v.view()
	if err := v.setConversation(&page, ci); err != nil {
		return err
	}
	page.CanvasActive = true

	// fetch messages so the full page renders correctly on deep link.
	itMsg, err := v.allMessagesOrEmpty(ctx, channelID)
	if err != nil {
		return err
	}
	page.Messages = itMsg

	return v.tmpl.ExecuteTemplate(w, "index.html", page)
}

// RenderCanvasContent writes the raw canvas HTML for channelID to w.
func (v *Viewer) RenderCanvasContent(ctx context.Context, channelID string, w io.Writer) error {
	ci, err := v.src.ChannelInfo(ctx, channelID)
	if err != nil {
		return err
	}
	if ci.Properties == nil || ci.Properties.Canvas.FileId == "" {
		return fs.ErrNotExist
	}
	fileID := ci.Properties.Canvas.FileId
	storage := v.src.Files()
	pth, err := fileByID(storage, fileID)
	if err != nil {
		return err
	}
	f, err := storage.FS().Open(pth)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

// ── HTTP handlers (thin wrappers) ────────────────────────────────────────────
// isHXRequest branching stays here; Render* methods are never called from
// within these HTMX branches.

func (v *Viewer) indexHandler(w http.ResponseWriter, r *http.Request) {
	if err := v.RenderIndex(r.Context(), w); err != nil {
		v.lg.ErrorContext(r.Context(), "indexHandler", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) channelHandler(w http.ResponseWriter, r *http.Request, id string) {
	if isHXRequest(r) {
		v.channelPartial(w, r, id)
		return
	}
	if err := v.RenderChannel(r.Context(), id, w); err != nil {
		lg := v.lg.With("in", "channelHandler", "channel", id)
		if errors.Is(err, source.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		lg.ErrorContext(r.Context(), "RenderChannel", "error", err)
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
			http.Redirect(w, r, v.rts.ThreadMessage(id, vts, structures.ThreadIDtoTS(ts)), http.StatusSeeOther)
		} else {
			// p refers to a message within the channel.
			// https: //ora600.slack.com/archives/DHMAB25DY/p1710063528879959
			lg.Debug("redirecting to channel message", "ts", ts)
			ts = structures.ThreadIDtoTS(ts)
			http.Redirect(w, r, v.rts.ChannelMessage(id, ts), http.StatusSeeOther)
		}
		return
	}
	lg.Debug("redirecting to thread message", "ts", ts)
	v.threadHandler(w, r, id)
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

	if isHXRequest(r) {
		v.threadPartial(w, r, id, ts)
		return
	}
	if err := v.RenderThread(r.Context(), id, ts, w); err != nil {
		lg := v.lg.With("in", "threadHandler", "channel", id, "thread", ts)
		lg.ErrorContext(r.Context(), "RenderThread", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

	if isHXRequest(r) && v.rts.Interactive() {
		if err := v.tmpl.ExecuteTemplate(w, "hx_user", userView{User: u, Interactive: true}); err != nil {
			lg.ErrorContext(ctx, "ExecuteTemplate", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := v.RenderUser(ctx, uid, w); err != nil {
		lg.ErrorContext(ctx, "RenderUser", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) canvasHandler(w http.ResponseWriter, r *http.Request, id string) {
	if isHXRequest(r) {
		v.canvasPartial(w, r, id)
		return
	}
	if err := v.RenderCanvas(r.Context(), id, w); err != nil {
		lg := v.lg.With("in", "canvasHandler", "channel", id)
		lg.ErrorContext(r.Context(), "RenderCanvas", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// canvasContentHandler streams the raw canvas HTML content for the given
// channel directly, without requiring the caller to know the filename.
func (v *Viewer) canvasContentHandler(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	lg := v.lg.With("in", "canvasContentHandler", "channel", id)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := v.RenderCanvasContent(ctx, id, w); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			lg.DebugContext(ctx, "canvas file not found", "error", err)
			http.NotFound(w, r)
			return
		}
		lg.ErrorContext(ctx, "RenderCanvasContent", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ── HTMX-only partial helpers (live mode) ────────────────────────────────────

func (v *Viewer) channelPartial(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	lg := v.lg.With("in", "channelPartial", "channel", id)

	it, err := v.src.AllMessages(ctx, id)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		lg.ErrorContext(ctx, "AllMessages", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
	page.Messages = it
	lg.DebugContext(ctx, "conversation", "id", id)
	if err := v.tmpl.ExecuteTemplate(w, "hx_conversation", page); err != nil {
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) threadPartial(w http.ResponseWriter, r *http.Request, id, ts string) {
	ctx := r.Context()
	lg := v.lg.With("in", "threadPartial", "channel", id, "thread", ts)

	itTm, err := v.src.AllThreadMessages(ctx, id, ts)
	if err != nil {
		lg.ErrorContext(ctx, "AllThreadMessages", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
	page.ThreadMessages = itTm
	page.ThreadID = ts
	lg.DebugContext(ctx, "Messages")
	if err := v.tmpl.ExecuteTemplate(w, "hx_thread", page); err != nil {
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err, "template", "hx_thread")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (v *Viewer) canvasPartial(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	lg := v.lg.With("in", "canvasPartial", "channel", id)

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
	if err := v.tmpl.ExecuteTemplate(w, "hx_canvas", page); err != nil {
		lg.ErrorContext(ctx, "ExecuteTemplate", "error", err, "template", "hx_canvas")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ── Remaining HTTP-only handlers ─────────────────────────────────────────────

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

// ── Shared helpers ────────────────────────────────────────────────────────────

// messageView is the data passed to the render_message template.
// It wraps a slack.Message with the channel ID needed to render threading
// context (reply-to anchor links).  Set ChannelID to an empty string to
// suppress the reply banner (used in the thread panel, where the parent
// message is on a different page).
type messageView struct {
	Msg         slack.Message
	ChannelID   string
	Interactive bool
}

type mainView struct {
	channels
	Name            string
	Type            string
	Interactive     bool
	Messages        iter.Seq2[slack.Message, error]
	ThreadMessages  iter.Seq2[slack.Message, error]
	ThreadID        string
	Conversation    slack.Channel
	User            *slack.User
	Alias           string // conversation alias
	AliasError      string
	CanAlias        bool // if true, alias can be set for the channel
	CanvasActive    bool // true when the canvas tab is the active tab
	CanvasAvailable bool // true when the canvas file exists in storage
}

type aliaser interface {
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
		channels:    v.ch,
		Name:        filepath.Base(v.src.Name()),
		Type:        v.src.Type().String(),
		Interactive: v.rts.Interactive(),
		CanAlias:    supportsAlias,
	}
}

type aliasAction int

const (
	aliasDelete aliasAction = iota
	aliasSet
)

func (v *Viewer) aliaser() (aliaser, bool) {
	a, ok := v.src.(aliaser)
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

func isHXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// isInvalid returns true if the provided path component is not web-safe.
func isInvalid(pcomp string) bool {
	return strings.Contains(pcomp, "..") || strings.HasPrefix(pcomp, "~") || strings.Contains(pcomp, "/") || strings.Contains(pcomp, "\\")
}

func emptyMessages() iter.Seq2[slack.Message, error] {
	return func(func(slack.Message, error) bool) {}
}

func (v *Viewer) allMessagesOrEmpty(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	it, err := v.src.AllMessages(ctx, channelID)
	if errors.Is(err, source.ErrNotFound) {
		return emptyMessages(), nil
	}
	return it, err
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

func (v *Viewer) canAlias() bool {
	_, ok := v.aliaser()
	return ok
}
