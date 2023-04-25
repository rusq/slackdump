// Package obfuscate obfuscates a slackdump chunk recording.  It provides
// deterministic obfuscation of IDs, so that the users within the obfuscated
// file will have a consistent IDs. But the same file obfuscated multiple
// times will have different IDs.  The text is replaced with the randomness of
// the same size + a random addition.
package obfuscate

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"hash"
	"io"
	"math/rand"
	"runtime/trace"
	"strings"
	"time"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/slack-go/slack"
)

type doOpts struct {
	seed int64
}

type Option func(*doOpts)

// WithSeed allows you to specify the seed for the random number generator.
func WithSeed(seed int64) Option {
	return func(opts *doOpts) {
		opts.seed = seed
	}
}

func Do(ctx context.Context, w io.Writer, r io.Reader, options ...Option) error {
	_, task := trace.NewTask(ctx, "obfuscate.Do")
	defer task.End()

	var opts = doOpts{
		seed: time.Now().UnixNano(),
	}
	for _, optFn := range options {
		optFn(&opts)
	}
	rand.Seed(opts.seed)

	var (
		dec = json.NewDecoder(r)
		enc = json.NewEncoder(w)
		obf = obfuscator{
			hasher: sha256.New,
			salt:   randomStringExact(32),
		}
	)
	// obfuscation loop
	for {
		var e chunk.Chunk
		if err := dec.Decode(&e); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		trace.WithRegion(ctx, "obfuscate.Event", func() {
			obf.Chunk(&e)
		})
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	return nil
}

type obfuscator struct {
	hasher func() hash.Hash
	salt   string
}

func (o obfuscator) Chunk(c *chunk.Chunk) {
	c.ChannelID = o.ChannelID(c.ChannelID)
	switch c.Type {
	case chunk.CMessages:
		o.Messages(c.Messages...)
	case chunk.CThreadMessages:
		o.OneMessage(c.Parent)
		o.Messages(c.Messages...)
	case chunk.CFiles:
		o.OneMessage(c.Parent)
		o.Files(c.Files...)
		o.Channel(c.Channel)
	case chunk.CChannelInfo:
		o.Channel(c.Channel)
	case chunk.CUsers:
		o.Users(c.Users...)
	default:
		dlog.Panicf("unknown chunk type: %s", c.Type)
	}

}

// obfuscateManyMessages obfuscates a slice of messages.
func (o obfuscator) Messages(m ...slack.Message) {
	for i := range m {
		o.OneMessage(&m[i])
	}
}

const fileURLPrefix = "https://files.slack.com/"

func notNilFn(s string, fn func(string) string) string {
	if s != "" {
		s = fn(s)
	}
	return s
}

func (o obfuscator) OneMessage(m *slack.Message) {
	if m == nil {
		return
	}
	m.ClientMsgID = notNilFn(m.ClientMsgID, func(s string) string { return randomUUID() })
	m.Team = o.TeamID(m.Team)
	m.Channel = o.UserID(m.Channel)
	m.User = o.UserID(m.User)
	if m.Text != "" {
		m.Text = randomString(len(m.Text))
	}
	if m.Edited != nil {
		m.Edited.User = o.UserID(m.Edited.User)
	}
	if len(m.Blocks.BlockSet) > 0 {
		m.Blocks.BlockSet = nil // too much hassle to obfuscate
	}
	if len(m.Reactions) > 0 {
		o.Reactions(m.Reactions)
	}
	if len(m.Attachments) > 0 {
		m.Attachments = nil // too much hassle to obfuscate
	}
	if m.Topic != "" {
		m.Topic = randomString(len(m.Topic))
	}
	m.Metadata = slack.SlackMetadata{}
	m.ParentUserId = o.UserID(m.ParentUserId)
	m.Team = o.TeamID(m.Team)
	for i := range m.ReplyUsers {
		m.ReplyUsers[i] = o.UserID(m.ReplyUsers[i])
	}
	o.BotProfile(m.BotProfile)
	m.Inviter = o.UserID(m.Inviter)

	for i := range m.Files {
		o.OneFile(&m.Files[i])
	}
}

func (o obfuscator) Files(f ...slack.File) {
	for i := range f {
		o.OneFile(&f[i])
	}
}

func (o obfuscator) OneFile(f *slack.File) {
	if f == nil {
		return
	}
	ifnotnil := func(s string) string {
		if s != "" {
			if strings.HasPrefix(s, fileURLPrefix) {
				s = fileURLPrefix + randomString(len(s)-len(fileURLPrefix))
			} else {
				s = randomString(len(s))
			}
		}
		return s
	}
	fields := []*string{
		&f.URLPrivate,
		&f.URLPrivateDownload,
		&f.Permalink,
		&f.PermalinkPublic,
		&f.Thumb64,
		&f.Thumb80,
		&f.Thumb360,
		&f.Thumb360Gif,
		&f.Thumb480,
		&f.Thumb160,
		&f.Thumb720,
		&f.Thumb960,
		&f.Thumb1024,
	}
	for i := range fields {
		*fields[i] = ifnotnil(*fields[i])
	}
	f.Title = ifnotnil(f.Title)
	f.Name = ifnotnil(f.Name)
	f.Thumb360W = 0
	f.Thumb360H = 0
	f.Thumb480W = 0
	f.Thumb480H = 0
	f.Thumb720W = 0
	f.Thumb720H = 0
	f.Thumb960W = 0
	f.Thumb960H = 0
	f.Thumb1024W = 0
	f.Thumb1024H = 0
	f.OriginalW = 0
	f.OriginalH = 0
	f.InitialComment = slack.Comment{}
	f.User = o.UserID(f.User)
	f.ID = o.FileID(f.ID)
}

// randomString returns a random string of length n + random number [0,40).
func randomString(n int) string {
	return rndstr(n, rand.Intn(40))
}

// randomStringExact returns a random string of length n.
func randomStringExact(n int) string {
	return rndstr(n, 0)
}

// rndstr returns a random string of length base+add.
func rndstr(base int, add int) string {
	var (
		b   = make([]byte, base+add)
		src = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 ")
	)
	for i := range b {
		b[i] = src[rand.Intn(len(src))]
	}
	return string(b)
}

// randomUUID returns a random UUID.
func randomUUID() string {
	var (
		b   = make([]byte, 36)
		src = []byte("0123456789abcdef")
	)
	for i := range b {
		switch i {
		case 8, 13, 18, 23:
			b[i] = '-'
		default:
			b[i] = src[rand.Intn(len(src))]
		}
	}
	return string(b)
}

func (o obfuscator) Reactions(r []slack.ItemReaction) {
	for i := range r {
		r[i].Name = randomStringExact(len(r[i].Name))
		for j := range r[i].Users {
			r[i].Users[j] = o.UserID(r[i].Users[j])
		}
	}
}

func (o obfuscator) Channel(c *slack.Channel) {
	if c == nil {
		return
	}
	c.ID = o.ChannelID(c.ID)
	c.Creator = o.UserID(c.Creator)
	c.Name = o.ID("", c.Name)
	c.NameNormalized = o.ID("", c.NameNormalized)

	c.Purpose.Value = randomStringExact(len(c.Purpose.Value))
	c.Purpose.Creator = o.UserID(c.Purpose.Creator)

	c.Topic.Value = randomStringExact(len(c.Topic.Value))
	c.Topic.Creator = o.UserID(c.Topic.Creator)

	for i := range c.Members {
		c.Members[i] = o.UserID(c.Members[i])
	}
}

func (o obfuscator) Users(uu ...slack.User) {
	for i := range uu {
		o.User(&uu[i])
	}
}

// TODO: test
func (o obfuscator) User(u *slack.User) {
	if u == nil {
		return
	}
	u.ID = o.UserID(u.ID)
	u.Name = o.ID("", u.Name)
	u.RealName = randomStringExact(len(u.RealName))
	u.Profile.DisplayName = randomStringExact(len(u.Profile.DisplayName))
	u.Profile.DisplayNameNormalized = randomStringExact(len(u.Profile.DisplayNameNormalized))
	u.Profile.RealName = randomStringExact(len(u.Profile.RealName))
	u.Profile.RealNameNormalized = randomStringExact(len(u.Profile.RealNameNormalized))
	u.Profile.Email = randomStringExact(len(u.Profile.Email))
	u.Profile.Image24 = randomStringExact(len(u.Profile.Image24))
	u.Profile.Image32 = randomStringExact(len(u.Profile.Image32))
	u.Profile.Image48 = randomStringExact(len(u.Profile.Image48))
	u.Profile.Image72 = randomStringExact(len(u.Profile.Image72))
	u.Profile.Image192 = randomStringExact(len(u.Profile.Image192))
	u.Profile.Image512 = randomStringExact(len(u.Profile.Image512))
	u.Profile.ImageOriginal = randomStringExact(len(u.Profile.ImageOriginal))
	u.Profile.StatusText = randomStringExact(len(u.Profile.StatusText))
	u.Profile.StatusEmoji = randomStringExact(len(u.Profile.StatusEmoji))
	u.Profile.StatusExpiration = 0
	u.Profile.Team = o.TeamID(u.Profile.Team)
}

func (o *obfuscator) BotProfile(bp *slack.BotProfile) {
	if bp == nil {
		return
	}
	bp.ID = o.BotID(bp.ID)
	bp.Deleted = false
	bp.Name = randomStringExact(len(bp.Name))
	bp.Updated = 0
	bp.AppID = o.AppID(bp.AppID)
	bp.TeamID = o.TeamID(bp.TeamID)
	bp.Icons.Image36 = randomStringExact(len(bp.Icons.Image36))
	bp.Icons.Image48 = randomStringExact(len(bp.Icons.Image48))
	bp.Icons.Image72 = randomStringExact(len(bp.Icons.Image72))
}
