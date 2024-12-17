package export

import (
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/structures"
)

// ExportMessage is the slack.Message with additional fields usually found in
// slack exports.
type ExportMessage struct {
	*slack.Msg

	// additional fields not defined by the slack library, but present
	// in slack exports
	UserTeam        string             `json:"user_team,omitempty"`
	SourceTeam      string             `json:"source_team,omitempty"`
	UserProfile     *ExportUserProfile `json:"user_profile,omitempty"`
	ReplyUsersCount int                `json:"reply_users_count,omitempty"`
	slackdumpTime   time.Time          `json:"-"` // to speedup sorting
}

type ExportUserProfile struct {
	AvatarHash        string `json:"avatar_hash"`
	Image72           string `json:"image_72"`
	FirstName         string `json:"first_name"`
	RealName          string `json:"real_name"`
	DisplayName       string `json:"display_name"`
	Team              string `json:"team"`
	Name              string `json:"name"`
	IsRestricted      bool   `json:"is_restricted"`
	IsUltraRestricted bool   `json:"is_ultra_restricted"`
}

func (em ExportMessage) Time() time.Time {
	if em.slackdumpTime.IsZero() {
		ts, _ := structures.ParseSlackTS(em.Timestamp)
		return ts
	}
	return em.slackdumpTime
}
