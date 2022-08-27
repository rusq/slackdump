//go:build exclude

// Package mm is an experimental package to add support to MM.  It is
// currently unused.
package mm

type Entry struct {
	Type          string        `json:"type"`
	Version       int64         `json:"version,omitempty"`
	Post          Post          `json:"post,omitempty"`
	Channel       Channel       `json:"channel,omitempty"`
	User          User          `json:"user"`
	DirectChannel DirectChannel `json:"direct_channel"`
	DirectPost    DirectPost    `json:"direct_post"`
}

type DirectPost struct {
	ChannelMembers []string      `json:"channel_members"`
	User           string        `json:"user"`
	Message        string        `json:"message"`
	Props          Props         `json:"props"`
	CreateAt       int64         `json:"create_at"`
	FlaggedBy      interface{}   `json:"flagged_by"`
	Reactions      interface{}   `json:"reactions"`
	Replies        []interface{} `json:"replies"`
	Attachments    []interface{} `json:"attachments"`
}

type Props struct {
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	ID         int64       `json:"id"`
	Fallback   string      `json:"fallback"`
	Color      string      `json:"color"`
	Pretext    string      `json:"pretext"`
	AuthorName string      `json:"author_name"`
	AuthorLink string      `json:"author_link"`
	AuthorIcon string      `json:"author_icon"`
	Title      string      `json:"title"`
	TitleLink  string      `json:"title_link"`
	Text       string      `json:"text"`
	Fields     interface{} `json:"fields"`
	ImageURL   string      `json:"image_url"`
	ThumbURL   string      `json:"thumb_url"`
	Footer     string      `json:"footer"`
	FooterIcon string      `json:"footer_icon"`
	Ts         string      `json:"ts"`
	Path       string      `json:"path"`
}

type DirectChannel struct {
	Members     []string    `json:"members"`
	FavoritedBy interface{} `json:"favorited_by"`
	Header      string      `json:"header"`
}

type User struct {
	Username    string      `json:"username"`
	Email       string      `json:"email"`
	AuthService interface{} `json:"auth_service"`
	Nickname    string      `json:"nickname"`
	FirstName   string      `json:"first_name"`
	LastName    string      `json:"last_name"`
	Position    string      `json:"position"`
	Roles       string      `json:"roles"`
	Locale      interface{} `json:"locale"`
	Teams       []Team      `json:"teams"`
}

type Team struct {
	Name     string        `json:"name"`
	Roles    string        `json:"roles"`
	Channels []interface{} `json:"channels"`
}

type Channel struct {
	Team        string `json:"team"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Header      string `json:"header"`
	Purpose     string `json:"purpose"`
}

type Post struct {
	Team        string       `json:"team"`
	Channel     string       `json:"channel"`
	User        string       `json:"user"`
	Message     string       `json:"message"`
	Props       Props        `json:"props"`
	CreateAt    int64        `json:"create_at"`
	Replies     []Reply      `json:"replies"`
	Attachments []Attachment `json:"attachments"`
}

type Reply struct {
	User        string       `json:"user"`
	Message     string       `json:"message"`
	CreateAt    int64        `json:"create_at"`
	Attachments []Attachment `json:"attachments"`
}
