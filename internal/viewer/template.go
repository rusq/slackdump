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
	"strings"
	"time"

	"github.com/rusq/slack"

	st "github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/source"
)

//go:embed templates
var fsys embed.FS

func initTemplates(v *Viewer) {
	tmpl := template.Must(template.New("").Funcs(
		template.FuncMap{
			"channelname":      v.channelDisplayName,
			"channelurl":       v.rts.Channel,
			"channelmsgurl":    v.rts.ChannelMessage,
			"threadurl":        v.rts.Thread,
			"threadmsgurl":     v.rts.ThreadMessage,
			"userurl":          v.rts.User,
			"profileurl":       v.profileURL,
			"canvasurl":        v.rts.Canvas,
			"canvascontenturl": v.rts.CanvasContent,
			"staticasset":      v.rts.StaticAsset,
			"chlink": func(ch slack.Channel, interactive bool) channelLinkView {
				return channelLinkView{Channel: ch, Interactive: interactive}
			},
			"userview": func(user *slack.User, interactive bool) userView {
				return userView{User: user, Interactive: interactive}
			},
			"staticuserview":  v.staticUserView,
			"is_app_msg":      isAppMsg,
			"is_user_msg":     isUserMsg,
			"displayname":     v.um.DisplayName,
			"username":        v.username, // username returns the username for the message
			"userpic":         v.userpic,  // userpic returns the userpic for the user
			"time":            localtime,
			"rendertext":      func(s string) string { return v.r.RenderText(context.Background(), s) },            // render message text
			"render":          func(m slack.Message) template.HTML { return v.r.Render(context.Background(), &m) }, // render message
			"is_thread_start": func(m slack.Message) bool { return st.IsThreadStart(&m) },
			"canvas_present":  func(ch slack.Channel) bool { return ch.Properties != nil && ch.Properties.Canvas.FileId != "" },
			"msgview": func(channelID string, m slack.Message) messageView {
				return messageView{Msg: m, ChannelID: channelID, Interactive: v.rts.Interactive()}
			},
		},
	).ParseFS(fsys, "templates/*.html"))
	v.tmpl = tmpl
}

type channelLinkView struct {
	Channel     slack.Channel
	Interactive bool
}

type userView struct {
	User        *slack.User
	Interactive bool
	TargetID    string
	CloseHref   string
}

func (v *Viewer) channelDisplayName(ch slack.Channel) template.HTML {
	name := v.um.ChannelName(ch)
	alias, ok, err := v.alias(ch.ID)
	if err != nil || !ok || alias == "" {
		return template.HTML(template.HTMLEscapeString(name))
	}
	archived := ""
	if ch.IsArchived {
		archived = " (archived)"
	}
	var buf strings.Builder
	buf.WriteString(template.HTMLEscapeString(st.ChannelPrefix(ch)))
	buf.WriteString("<em>")
	buf.WriteString(template.HTMLEscapeString(alias))
	buf.WriteString("</em>")
	buf.WriteString(template.HTMLEscapeString(archived))
	return template.HTML(buf.String())
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

const emptyAvatar = "48x48.gif"

func (v *Viewer) userpic(userID string) string {
	if userID == "" {
		return v.rts.StaticAsset(emptyAvatar)
	}
	user, ok := v.um[userID]
	if !v.rts.Interactive() {
		// Static mode: use local avatar when it was actually downloaded.
		if ok && user.Profile.ImageOriginal != "" {
			uid, filename := source.AvatarParams(user)
			if _, err := v.src.Avatars().File(uid, filename); err == nil {
				return v.rts.Avatar(userID, filename)
			}
		}
		// Local avatar unavailable; fall through to CDN URL so the browser
		// can still fetch it, or use the empty-avatar placeholder.
		if ok && user.Profile.Image48 != "" {
			return user.Profile.Image48
		}
		return v.rts.StaticAsset(emptyAvatar)
	}
	// Live mode: use Slack CDN URL directly.
	if ok && user.Profile.Image48 != "" {
		return user.Profile.Image48
	}
	slog.Debug("userpic not found", "user", userID)
	return v.rts.StaticAsset(emptyAvatar)
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

func (v *Viewer) profileURL(m slack.Message) string {
	if v.rts.Interactive() {
		return v.rts.User(m.User)
	}
	return "#" + profileTargetID(m)
}

func (v *Viewer) staticUserView(m slack.Message) userView {
	return userView{
		User:      v.um[m.User],
		TargetID:  profileTargetID(m),
		CloseHref: "#" + safeAnchorID(m.Timestamp),
	}
}

func profileTargetID(m slack.Message) string {
	return "user-profile-" + safeAnchorID(m.User+"-"+m.Timestamp)
}

func safeAnchorID(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

func isAppMsg(m slack.Message) bool {
	sender := msgsender(m)
	return sender == sApp || sender == sBot
}

func isUserMsg(m slack.Message) bool {
	return msgsender(m) == sUser
}
