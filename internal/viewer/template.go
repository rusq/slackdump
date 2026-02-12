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
	"embed"
	"html/template"
	"log/slog"
	"time"

	"github.com/rusq/slack"

	st "github.com/rusq/slackdump/v4/internal/structures"
)

//go:embed templates
var fsys embed.FS

func initTemplates(v *Viewer) {
	tmpl := template.Must(template.New("").Funcs(
		template.FuncMap{
			"rendername":      v.um.ChannelName,
			"is_app_msg":      isAppMsg,
			"is_user_msg":     isUserMsg,
			"displayname":     v.um.DisplayName,
			"username":        v.username, // username returns the username for the message
			"userpic":         v.userpic,  // userpic returns the userpic for the user
			"time":            localtime,
			"rendertext":      func(s string) string { return v.r.RenderText(context.Background(), s) },            // render message text
			"render":          func(m slack.Message) template.HTML { return v.r.Render(context.Background(), &m) }, // render message
			"is_thread_start": func(m slack.Message) bool { return st.IsThreadStart(&m) },
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

func msgsender(m slack.Message) sender {
	if m.BotID != "" {
		if m.Username != "" {
			return sApp
		}
		return sBot
	}
	if m.User != "" {
		return sUser
	}
	return sUnknown
}

const emptyAvatar = "/static/48x48.gif"

func (v *Viewer) userpic(userID string) string {
	if userID == "" {
		return emptyAvatar
	}
	user, ok := v.um[userID]
	if ok && user.Profile.Image48 != "" {
		return user.Profile.Image48
	}
	slog.Debug("userpic not found", "user", userID)

	return emptyAvatar
}

func (v *Viewer) username(m slack.Message) (name string) {
	switch msgsender(m) {
	case sUser:
		return v.um.DisplayName(m.User)
	case sBot:
		name := m.BotID
		if m.BotProfile != nil {
			name = m.BotProfile.Name
		}
		user := "Unknown user"
		if m.User != "" {
			user = v.um.DisplayName(m.User)
		}
		return user + " via " + name + " (bot)"
	case sApp:
		return m.Username + " (app)"
	case sUnknown:
		return "<UNKNOWN>"
	default:
		panic("unhandled sender type")
	}
}

func isAppMsg(m slack.Message) bool {
	sender := msgsender(m)
	return sender == sApp || sender == sBot
}

func isUserMsg(m slack.Message) bool {
	return msgsender(m) == sUser
}
