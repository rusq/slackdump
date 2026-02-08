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
	"errors"
	"hash"
	"io"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime/trace"
	"strings"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
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

func newObfuscator(rng *rand.Rand) obfuscator {
	o := obfuscator{
		hasher: sha256.New,
		rng:    rng,
	}
	o.salt = o.randomStringExact(32)
	return o
}

// Do obfuscates the slackdump chunk recording from r, writing the obfuscated
// chunks to w.
func Do(ctx context.Context, w io.Writer, r io.Reader, options ...Option) error {
	_, task := trace.NewTask(ctx, "obfuscate.Do")
	defer task.End()

	opts := doOpts{
		seed: time.Now().UnixNano(),
	}
	for _, optFn := range options {
		optFn(&opts)
	}
	rng := rand.New(rand.NewSource(opts.seed))
	obf := newObfuscator(rng)
	return obfuscate(ctx, obf, w, r)
}

func obfuscate(ctx context.Context, obf obfuscator, w io.Writer, r io.Reader) error {
	signal.Reset(os.Interrupt)
	var (
		dec = json.NewDecoder(r)
		enc = json.NewEncoder(w)
	)
	// obfuscation loop
	for {
		var e chunk.Chunk
		if err := dec.Decode(&e); err != nil {
			if errors.Is(err, io.EOF) {
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
	rng    *rand.Rand
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
	case chunk.CChannelUsers:
		o.ChannelUsers(c.ChannelUsers)
	case chunk.CUsers:
		o.Users(c.Users...)
	case chunk.CChannels:
		o.Channels(c.Channels...)
	case chunk.CWorkspaceInfo:
		o.WorkspaceInfo(c.WorkspaceInfo)
	default:
		log.Panicf("unknown chunk type: %s", c.Type)
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
	m.ClientMsgID = notNilFn(m.ClientMsgID, func(s string) string { return o.randomUUID() })
	m.Team = o.TeamID(m.Team)
	m.Channel = o.UserID(m.Channel)
	m.User = o.UserID(m.User)
	if m.Text != "" {
		m.Text = o.randomString(len(m.Text))
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
		m.Topic = o.randomString(len(m.Topic))
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
				s = fileURLPrefix + o.randomString(len(s)-len(fileURLPrefix))
			} else {
				s = o.randomString(len(s))
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
func (o obfuscator) randomString(n int) string {
	return o.rndstr(n, o.rng.Intn(40))
}

// randomStringExact returns a random string of length n.
func (o obfuscator) randomStringExact(n int) string {
	return o.rndstr(n, 0)
}

// rndstr returns a random string of length base+add.
func (o obfuscator) rndstr(base int, add int) string {
	var (
		b   = make([]byte, base+add)
		src = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 ")
	)
	for i := range b {
		b[i] = src[o.rng.Intn(len(src))]
	}
	return string(b)
}

// randomUUID returns a random UUID.
func (o obfuscator) randomUUID() string {
	var (
		b   = make([]byte, 36)
		src = []byte("0123456789abcdef")
	)
	for i := range b {
		switch i {
		case 8, 13, 18, 23:
			b[i] = '-'
		default:
			b[i] = src[o.rng.Intn(len(src))]
		}
	}
	return string(b)
}

func (o obfuscator) Reactions(r []slack.ItemReaction) {
	for i := range r {
		r[i].Name = o.randomStringExact(len(r[i].Name))
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
	o.OneMessage(c.Latest)

	c.Purpose.Value = o.randomStringExact(len(c.Purpose.Value))
	c.Purpose.Creator = o.UserID(c.Purpose.Creator)

	c.Topic.Value = o.randomStringExact(len(c.Topic.Value))
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
	u.RealName = o.randomStringExact(len(u.RealName))
	u.TeamID = o.TeamID(u.TeamID)
	o.Profile(&u.Profile)
}

func (o obfuscator) Profile(p *slack.UserProfile) {
	if p == nil {
		return
	}
	p.DisplayName = o.randomStringExact(len(p.DisplayName))
	p.DisplayNameNormalized = o.randomStringExact(len(p.DisplayNameNormalized))
	p.RealName = o.randomStringExact(len(p.RealName))
	p.RealNameNormalized = o.randomStringExact(len(p.RealNameNormalized))
	p.FirstName = o.randomString(len(p.FirstName))
	p.LastName = o.randomString(len(p.LastName))
	p.Email = o.randomStringExact(len(p.Email))
	p.Skype = o.randomStringExact(len(p.Skype))
	p.Phone = o.randomStringExact(len(p.Phone))
	p.Image24 = o.randomStringExact(len(p.Image24))
	p.Image32 = o.randomStringExact(len(p.Image32))
	p.Image48 = o.randomStringExact(len(p.Image48))
	p.Image72 = o.randomStringExact(len(p.Image72))
	p.Image192 = o.randomStringExact(len(p.Image192))
	p.Image512 = o.randomStringExact(len(p.Image512))
	p.ImageOriginal = o.randomStringExact(len(p.ImageOriginal))
	p.StatusText = o.randomStringExact(len(p.StatusText))
	p.StatusEmoji = o.randomStringExact(len(p.StatusEmoji))
	p.StatusExpiration = 0
	p.Team = o.TeamID(p.Team)
}

func (o obfuscator) BotProfile(bp *slack.BotProfile) {
	if bp == nil {
		return
	}
	bp.ID = o.BotID(bp.ID)
	bp.Deleted = false
	bp.Name = o.randomStringExact(len(bp.Name))
	bp.Updated = 0
	bp.AppID = o.AppID(bp.AppID)
	bp.TeamID = o.TeamID(bp.TeamID)
	bp.Icons.Image36 = o.randomStringExact(len(bp.Icons.Image36))
	bp.Icons.Image48 = o.randomStringExact(len(bp.Icons.Image48))
	bp.Icons.Image72 = o.randomStringExact(len(bp.Icons.Image72))
}

func (o obfuscator) ChannelUsers(cu []string) {
	for i := range cu {
		cu[i] = o.UserID(cu[i])
	}
}

func (o obfuscator) Channels(c ...slack.Channel) {
	for i := range c {
		o.Channel(&c[i])
	}
}

func (o obfuscator) WorkspaceInfo(wi *slack.AuthTestResponse) {
	wi.BotID = o.BotID(wi.BotID)
	wi.TeamID = o.TeamID(wi.TeamID)
	wi.UserID = o.UserID(wi.UserID)
	wi.URL = o.randomString(len(wi.URL))
	wi.Team = o.randomString(len(wi.Team))
	wi.User = o.randomString(len(wi.User))
	wi.EnterpriseID = o.EnterpriseID(wi.EnterpriseID)
}
