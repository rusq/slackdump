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
	var tmpl = template.Must(template.New("").Funcs(
		template.FuncMap{
			"rendername":      v.um.ChannelName,
			"displayname":     v.um.DisplayName,
			"time":            localtime,
			"rendertext":      func(s string) template.HTML { return v.r.RenderText(context.Background(), s) },     // render message text
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
