// Package obfuscate obfuscates a slackdump chunk recording.  It provides
// deterministic obfuscation of IDs, so that the users within the obfuscated
// file will have a consistent IDs. But the same file obfuscated multiple
// times will have different IDs.  The text is replaced with the randomness of
// the same size + a random addition.
package obfuscate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	c.ChannelID = o.ID("C", c.ChannelID)
	switch c.Type {
	case chunk.CMessages:
		o.Messages(c.Messages...)
	case chunk.CThreadMessages:
		o.OneMessage(c.Parent)
		o.Messages(c.Messages...)
	case chunk.CFiles:
		o.OneMessage(c.Parent)
		o.Files(c.Files...)
	case chunk.CChannelInfo:
		o.Channel(c.Channel)
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

const filePrefix = "https://files.slack.com/"

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
	m.Team = o.ID("T", m.Team)
	m.Channel = o.ID("C", m.Channel)
	m.User = o.ID("U", m.User)
	if m.Text != "" {
		m.Text = randomString(len(m.Text))
	}
	if m.Edited != nil {
		m.Edited.User = o.ID("U", m.Edited.User)
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
	if m.ParentUserId != "" {
		m.ParentUserId = o.ID("U", m.ParentUserId)
	}
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
			if strings.HasPrefix(s, filePrefix) {
				s = filePrefix + randomString(len(s)-len(filePrefix))
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
	f.User = o.ID("U", f.User)
	f.ID = o.ID("F", f.ID)
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

// ID obfuscates an ID.
func (o obfuscator) ID(prefix string, id string) string {
	if id == "" {
		return ""
	}
	h := o.hasher()
	if _, err := h.Write([]byte(o.salt + id)); err != nil {
		panic(err)
	}
	return prefix + strings.ToUpper(hex.EncodeToString(h.Sum(nil)))[:len(id)-1]
}

func (o obfuscator) Reactions(r []slack.ItemReaction) {
	for i := range r {
		r[i].Name = randomStringExact(len(r[i].Name))
		for j := range r[i].Users {
			r[i].Users[j] = o.ID("U", r[i].Users[j])
		}
	}
}

func (o obfuscator) Channel(c *slack.Channel) {
	if c == nil {
		return
	}
	c.ID = o.ID("C", c.ID)
	c.Creator = o.ID("U", c.Creator)
	c.Name = o.ID("", c.Name)
	c.NameNormalized = o.ID("", c.NameNormalized)
	c.Purpose.Value = randomStringExact(len(c.Purpose.Value))
	c.Topic.Value = randomStringExact(len(c.Topic.Value))
	for i := range c.Members {
		c.Members[i] = o.ID("U", c.Members[i])
	}
}
