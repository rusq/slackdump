package expproc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"
	"sort"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/osext"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
)

type Transform struct {
	srcdir string       // source directory with chunks.
	fsa    fsadapter.FS // target file system adapter.
	ids    chan string  // channel used to pass channel IDs to the worker.
	done   chan struct{}
	users  []slack.User // list of users.
	err    chan error   // error channel used to propagate errors to the main thread.
}

type TfOption func(*Transform)

func WithBufferSize(n int) TfOption {
	return func(t *Transform) {
		t.ids = make(chan string, n)
	}
}

// WithUsers allows to pass a list of users to the transformer.
func WithUsers(users []slack.User) TfOption {
	return func(t *Transform) {
		t.users = users
	}
}

// NewTransform creates a new Transform instance.
func NewTransform(ctx context.Context, fsa fsadapter.FS, chunkdir string, tfopt ...TfOption) (*Transform, error) {
	if err := osext.DirExists(chunkdir); err != nil {
		return nil, fmt.Errorf("chunk directory %s does not exist: %w", chunkdir, err)
	}
	t := &Transform{
		srcdir: chunkdir,
		fsa:    fsa,
		ids:    make(chan string),
		done:   make(chan struct{}),
		err:    make(chan error, 1),
	}
	for _, opt := range tfopt {
		opt(t)
	}
	return t, nil
}

// Start starts the Transform processor.
func (t *Transform) StartWithUsers(ctx context.Context, users []slack.User) error {
	if users == nil {
		return errors.New("users list is nil")
	}
	t.users = users
	go t.worker(ctx)
	return nil
}

// Start starts the Transform processor, the users must have been initialised
// with the WithUsers option.  Otherwise, use StartWithUsers method.
func (t *Transform) Start(ctx context.Context) error {
	if t.users == nil {
		return errors.New("internal error: users not initialised")
	}
	go t.worker(ctx)
	return nil
}

// OnFinish is the function that should be passed to the Channel processor.
// It will not block if the internal buffer is full.  Buffer size can be
// set with the WithBufferSize option.
func (t *Transform) OnFinish(channelID string) error {
	select {
	case err := <-t.err:
		return err
	default:
		t.ids <- channelID
	}
	return nil
}

func (t *Transform) worker(ctx context.Context) {
	for id := range t.ids {
		if err := transform(ctx, t.fsa, t.srcdir, id, t.users); err != nil {
			t.err <- err
			continue
		}
	}
	close(t.done)
}

// Close closes the Transform processor.  It must be called once it is
// guaranteed that OnFinish will not be called anymore, otherwise the
// call to OnFinish will panic.
func (t *Transform) Close() error {
	close(t.ids)
	<-t.done
	return nil
}

// transform is the chunk file transformer.  It transforms the chunk file for
// the channel with ID into a slack export format, and attachments are placed
// into the relevant directory.  It expects the chunk file to be in the
// srcdir/id.json.gz file, and the attachments to be in the srcdir/id
// directory.
func transform(ctx context.Context, fsa fsadapter.FS, srcdir string, id string, users []slack.User) error {
	ctx, task := trace.NewTask(ctx, "transform")
	defer task.End()

	// load the chunk file
	f, err := openChunks(filepath.Join(srcdir, id+ext))
	if err != nil {
		return err
	}
	defer f.Close()
	// locate attachments
	// var hasAttachments bool
	// if hasAttachments, err = dirExists(filepath.Join(srcdir, id)); err != nil {
	// 	return err
	// }
	// transform the chunk file
	cf, err := chunk.FromReader(f)
	if err != nil {
		return err
	}

	ci, err := cf.ChannelInfo(id)
	if err != nil {
		return err
	}

	if err := writeMessages(ctx, fsa, cf, ci, users); err != nil {
		return err
	}

	return nil
}

func LoadUsers(ctx context.Context, dir string) ([]slack.User, error) {
	f, err := openChunks(filepath.Join(dir, "users"+ext))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	p, err := chunk.FromReader(f)
	if err != nil {
		return nil, err
	}
	users, err := p.AllUsers()
	if err != nil {
		return nil, err
	}
	return users, nil
}

func channelName(ch *slack.Channel) string {
	if ch.IsIM {
		return ch.ID
	}
	return ch.Name
}

func writeMessages(ctx context.Context, fsa fsadapter.FS, pl *chunk.File, ci *slack.Channel, users []slack.User) error {
	uidx := types.Users(users).IndexByID()
	dir := channelName(ci)
	var (
		prevDt string         // previous date
		wc     io.WriteCloser // current file
		enc    *json.Encoder  // current encoder
	)
	if err := pl.Sorted(ctx, false, func(ts time.Time, m *slack.Message) error {
		date := ts.Format("2006-01-02")
		if date != prevDt || prevDt == "" {
			if wc != nil {
				if err := writeJSONFooter(wc); err != nil {
					return err
				}
				if err := wc.Close(); err != nil {
					return err
				}
			}
			var err error
			wc, err = fsa.Create(filepath.Join(dir, date+".json"))
			if err != nil {
				return err
			}
			if err := writeJSONHeader(wc); err != nil {
				return err
			}
			prevDt = date
			enc = json.NewEncoder(wc)
			enc.SetIndent("", "  ")
		} else {
			wc.Write([]byte(",\n"))
		}
		var thread []slack.Message
		if m.ThreadTimestamp == m.Timestamp {
			// get the thread for the initial thread message only.
			var err error
			thread, err = pl.AllThreadMessages(ci.ID, m.ThreadTimestamp)
			if err != nil {
				return err
			}
		}
		// transform the message
		em := toExportMessage(m, thread, uidx[m.User])

		return enc.Encode(em)
	}); err != nil {
		return err
	}
	// write the last footer
	if wc != nil {
		if err := writeJSONFooter(wc); err != nil {
			return err
		}
		if err := wc.Close(); err != nil {
			return err
		}
	}
	return nil
}

func toExportMessage(m *slack.Message, thread []slack.Message, user *slack.User) *export.ExportMessage {
	em := export.ExportMessage{
		Msg:        &m.Msg,
		UserTeam:   m.Team,
		SourceTeam: m.Team,
	}
	if user != nil && !user.IsBot {
		em.UserProfile = &export.ExportUserProfile{
			AvatarHash:        "",
			Image72:           user.Profile.Image72,
			FirstName:         user.Profile.FirstName,
			RealName:          user.Profile.RealName,
			DisplayName:       user.Profile.DisplayName,
			Team:              user.Profile.Team,
			Name:              user.Name,
			IsRestricted:      user.IsRestricted,
			IsUltraRestricted: user.IsUltraRestricted,
		}
	}
	if len(thread) > 0 {
		em.Replies = make([]slack.Reply, 0, len(thread))
		for _, rm := range thread {
			em.Replies = append(em.Replies, slack.Reply{
				User:      rm.User,
				Timestamp: rm.Timestamp,
			})
			em.ReplyUsers = append(em.ReplyUsers, rm.User)
		}
		sort.Slice(em.Msg.Replies, func(i, j int) bool {
			tsi, err := structures.TS2int(em.Msg.Replies[i].Timestamp)
			if err != nil {
				return false
			}
			tsj, err := structures.TS2int(em.Msg.Replies[j].Timestamp)
			if err != nil {
				return false
			}
			return tsi < tsj
		})
		makeUniqueStrings(&em.ReplyUsers)
		em.ReplyUsersCount = len(em.ReplyUsers)
	}
	return &em
}

func makeUniqueStrings(ss *[]string) {
	if len(*ss) == 0 {
		return
	}
	sort.Strings(*ss)
	i := 0
	for _, s := range *ss {
		if s != (*ss)[i] {
			i++
			(*ss)[i] = s
		}
	}
	*ss = (*ss)[:i+1]
}

func writeJSONHeader(w io.Writer) error {
	_, err := io.WriteString(w, "[\n")
	return err
}

func writeJSONFooter(w io.Writer) error {
	_, err := io.WriteString(w, "\n]\n")
	return err
}

func openChunks(filename string) (io.ReadSeekCloser, error) {
	if fi, err := os.Stat(filename); err != nil {
		return nil, err
	} else if fi.IsDir() {
		return nil, errors.New("chunk file is a directory")
	} else if fi.Size() == 0 {
		return nil, errors.New("chunk file is empty")
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tf, err := osext.UnGZIP(f)
	if err != nil {
		return nil, err
	}

	return osext.RemoveOnClose(tf, tf.Name()), nil
}
