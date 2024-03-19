package viewer

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"log/slog"
	"mime"
	"strings"
	"time"

	"github.com/rusq/slack"
	st "github.com/rusq/slackdump/v3/internal/structures"
)

//go:embed templates
var fsys embed.FS

func initTemplates(v *Viewer) {
	var tmpl = template.Must(template.New("").Funcs(
		template.FuncMap{
			"rendername":      v.um.ChannelName,
			"is_app_msg":      isAppMsg,
			"displayname":     v.um.DisplayName,
			"username":        v.username, // username returns the username for the message
			"time":            localtime,
			"epoch":           epoch,
			"rendertext":      func(s string) template.HTML { return v.r.RenderText(context.Background(), s) },     // render message text
			"render":          func(m *slack.Message) template.HTML { return v.r.Render(context.Background(), m) }, // render message
			"is_thread_start": st.IsThreadStart,
			"mimetype":        mimetype,
		},
	).ParseFS(fsys, "templates/*.html"))
	v.tmpl = tmpl
}

func localtime(ts string) string {
	t, err := st.ParseSlackTS(ts)
	if err != nil {
		return ts
	}
	return t.Local().Format(time.DateTime)
}

func epoch(ts json.Number) string {
	if ts == "" {
		return ""
	}
	t, err := ts.Int64()
	if err != nil {
		slog.Debug("epoch Int64 error, trying float", "err", err, "ts", ts)
		tf, err := ts.Float64()
		if err != nil {
			slog.Debug("epoch Float64 error", "err", err, "ts", ts)
			return ts.String()
		}
		t = int64(tf)
	}
	return time.Unix(t, 0).Local().Format(time.DateTime)
}

type sender int

const (
	sUnknown sender = iota
	sUser
	sBot
	sApp
)

func msgsender(m *slack.Message) sender {
	if m.BotID != "" {
		if m.Username != "" {
			return sApp
		}
		if m.BotProfile != nil && m.BotProfile.Name != "" {
			return sBot
		}
	}
	if m.User != "" {
		return sUser
	}
	return sUnknown
}

func (v *Viewer) username(m *slack.Message) string {
	switch msgsender(m) {
	case sUser:
		return v.um.DisplayName(m.User)
	case sBot:
		return v.um.DisplayName(m.User) + " via " + m.BotProfile.Name + " (bot)"
	case sApp:
		return m.Username + " (app)"
	case sUnknown:
		return "<UNKNOWN>"
	default:
		panic("unhandled sender type")
	}
}

func isAppMsg(m *slack.Message) bool {
	return msgsender(m) == sApp
}

func mimetype(mt string) string {
	mm, _, err := mime.ParseMediaType(mt)
	if err != nil || mt == "" {
		slog.Debug("mimetype", "err", err, "mimetype", mt)
		return "application"
	}
	slog.Debug("mimetype", "t", mm, "mimetype", mt)
	t, _, found := strings.Cut(mm, "/")
	if !found {
		return "application"
	}
	return t
}
