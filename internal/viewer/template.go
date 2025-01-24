package viewer

import (
	"context"
	"embed"
	"html/template"
	"time"

	"github.com/rusq/slack"

	st "github.com/rusq/slackdump/v3/internal/structures"
)

//go:embed templates
var fsys embed.FS

func initTemplates(v *Viewer) {
	tmpl := template.Must(template.New("").Funcs(
		template.FuncMap{
			"rendername":      v.um.ChannelName,
			"is_app_msg":      isAppMsg,
			"displayname":     v.um.DisplayName,
			"username":        v.username, // username returns the username for the message
			"time":            localtime,
			"rendertext":      func(s string) string { return v.r.RenderText(context.Background(), s) },            // render message text
			"render":          func(m *slack.Message) template.HTML { return v.r.Render(context.Background(), m) }, // render message
			"is_thread_start": st.IsThreadStart,
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
