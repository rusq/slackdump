package expproc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime/trace"
	"sort"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/osext"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/rusq/slackdump/v2/types"
)

// Transform is a transformer that takes the chunks produced by the processor
// and transforms them into a Slack Export format.  It is sutable for async
// processing, in which case, OnFinalise function is passed to the processor,
// and the finalisation requests will be queued (up to a certain limit) and
// will be processed once Start or StartWithUsers is called.
//
// Please note, that transform requires users to be passed either through
// options or through StartWithUsers.  If users are not passed, the transform
// will fail.
//
// The asynchronous pattern to run the transform is as follows:
//
//  1. Create the transform instance.
//  2. Defer its Close method.
//  3. In goroutine: Start user processing, and in the same goroutine, after
//     all users are fetched, call [Transform.StartWithUsers], passing the
//     fetched users slice.
//  4. In another goroutine, start the Export Conversation processor, passsing
//     the transformer's OnFinalise function as the finaliser option.  It will
//     be called by export processor for each channel that was completed.
//
// TODO: Chunk channels and index generation here.
type Transform struct {
	srcdir string       // source directory with chunks.
	fsa    fsadapter.FS // target file system adapter.
	users  []slack.User // list of users.
	lg     logger.Interface

	start chan struct{}
	done  chan struct{}
	err   chan error  // error channel used to propagate errors to the main thread.
	ids   chan string // channel used to pass channel IDs to the worker.
}

// idsBufSz is the default size of the channel IDs buffer.  This is the number
// of channel IDs that will be queued without blocking before the Transform is
// started.
const idsBufSz = 100

// TfOption is a function that configures the Transform instance.
type TfOption func(*Transform)

// WithBufferSize sets the size of the channel IDs buffer.  This is the number
// of channel IDs that will be queued without blocking before the Transform is
// started.
func WithBufferSize(n int) TfOption {
	return func(t *Transform) {
		if n < 1 {
			n = idsBufSz
		}
		t.ids = make(chan string, n)
	}
}

// WithUsers allows to pass a list of users to the transform.
func WithUsers(users []slack.User) TfOption {
	return func(t *Transform) {
		t.users = users
	}
}

// NewTransform creates a new Transform instance.  The fsa is the filesystem
// adapter that holds the transformed data (output), chunkdir is the directory
// where the chunks, produced by processor, are stored.
func NewTransform(ctx context.Context, fsa fsadapter.FS, chunkdir string, tfopt ...TfOption) (*Transform, error) {
	if err := osext.DirExists(chunkdir); err != nil {
		return nil, fmt.Errorf("chunk directory %s does not exist: %w", chunkdir, err)
	}
	lg := logger.FromContext(ctx)
	t := &Transform{
		srcdir: chunkdir,
		fsa:    fsa,
		lg:     lg,
		start:  make(chan struct{}),
		done:   make(chan struct{}),
		ids:    make(chan string, idsBufSz),
		err:    make(chan error, 1),
	}
	for _, opt := range tfopt {
		opt(t)
	}
	go t.worker(ctx) // wont run until something is sent into start channel
	return t, nil
}

// WriteUsers writes the list of users to the file system adapter.
func (t *Transform) WriteUsers(users []slack.User) error {
	if users == nil {
		return errors.New("users list is nil")
	}
	t.lg.Debugln("transform: writing users")
	return t.writeUsers(users)
}

// writeUsers writes the list of users to the file system adapter.
func (t *Transform) writeUsers(users []slack.User) error {
	f, err := t.fsa.Create("users.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(users)
}

// Start starts the Transform processor with the provided list of users.
// Users are used to populate each message with the user profile, as per Slack
// original export format.
func (t *Transform) StartWithUsers(ctx context.Context, users []slack.User) error {
	if len(users) == 0 {
		return errors.New("users list is empty or nil")
	}
	t.users = users
	return t.Start(ctx)
}

func (t *Transform) hasUsers() bool {
	return len(t.users) > 0
}

// Start starts the Transform processor, the users must have been initialised
// with the WithUsers option.  Otherwise, use StartWithUsers method.
// If the processor is already started, it will return nil.
func (t *Transform) Start(ctx context.Context) error {
	t.lg.Debugln("transform: starting transform")
	if !t.hasUsers() {
		return errors.New("internal error: users not initialised")
	}

	t.start <- struct{}{}

	return nil
}

// OnFinalise is the function that should be passed to the Channel processor.
// It will not block if the internal buffer is full.  Buffer size can be
// set with the WithBufferSize option.  The caller is allowed to call OnFinalise
// even if the processor is not started, in which case the channel ID will
// be queued for processing once the processor is started.
func (t *Transform) OnFinalise(ctx context.Context, channelID string) error {
	t.lg.Debugln("transform: placing channel in the queue", channelID)
	select {
	case err := <-t.err:
		return err
	default:
		t.ids <- channelID
	}
	return nil
}

func (t *Transform) worker(ctx context.Context) {
	defer close(t.done)

	t.lg.Debugln("transform: worker waiting")
	<-t.start
	t.lg.Debugln("transform: worker started")
	for id := range t.ids {
		t.lg.Debugf("transform: transforming channel %s", id)
		if err := transform(ctx, t.fsa, t.srcdir, id, t.users); err != nil {
			t.lg.Debugf("transform: error transforming channel %s: %s", id, err)
			t.err <- err
			continue
		}
	}
}

// WriteIndex generates and writes the export index files.  It must be called
// once all transformations are done, because it might require to read channel
// files.
func (t *Transform) WriteIndex() error {
	t.lg.Debugln("transform: finalising transform")
	cd, err := chunk.OpenDir(t.srcdir)
	if err != nil {
		return fmt.Errorf("error opening chunk directory: %w", err)
	}
	wsp, err := cd.WorkspaceInfo()
	if err != nil {
		return fmt.Errorf("failed to get the workspace info: %w", err)
	}
	chans, err := cd.Channels() // this might read the channel files if it doesn't find the channels list chunks.
	if err != nil {
		return fmt.Errorf("error indexing channels: %w", err)
	}
	eidx, err := structures.MakeExportIndex(chans, t.users, wsp.UserID)
	if err != nil {
		return fmt.Errorf("error creating export index: %w", err)
	}
	if err := eidx.Marshal(t.fsa); err != nil {
		return fmt.Errorf("error writing export index: %w", err)
	}
	return nil
}

// Close closes the Transform processor.  It must be called once it is
// guaranteed that OnFinish will not be called anymore, otherwise the
// call to OnFinish will panic.
func (t *Transform) Close() error {
	t.lg.Debugln("transform: closing transform")
	close(t.ids)
	close(t.start)
	t.lg.Debugln("transform: waiting for workers to finish")
	<-t.done
	return nil
}

// transform is the chunk file transform.  It transforms the chunk file for
// the channel with ID into a slack export format, and attachments are placed
// into the relevant directory.  It expects the chunk file to be in the
// srcdir/id.json.gz file, and the attachments to be in the srcdir/id
// directory.
func transform(ctx context.Context, fsa fsadapter.FS, srcdir string, id string, users []slack.User) error {
	ctx, task := trace.NewTask(ctx, "transform")
	defer task.End()

	lg := logger.FromContext(ctx)
	trace.Logf(ctx, "input", "len(users)=%d", len(users))
	lg.Debugf("transforming channel %s, user len=%d", id, len(users))

	cd, err := chunk.OpenDir(srcdir)
	if err != nil {
		return fmt.Errorf("error opening chunk directory %q: %w", srcdir, err)
	}

	// load the chunk file
	cf, err := cd.Open(id)
	if err != nil {
		return fmt.Errorf("error opening chunk file %q: %w", id, err)
	}
	defer cf.Close()

	ci, err := cf.ChannelInfo(id)
	if err != nil {
		return fmt.Errorf("error reading channel info for %q: %w", id, err)
	}

	if err := writeMessages(ctx, fsa, cf, ci, users); err != nil {
		return err
	}

	return nil
}

// writeMessages writes the messages to the file system adapter.
func writeMessages(ctx context.Context, fsa fsadapter.FS, pl *chunk.File, ci *slack.Channel, users []slack.User) error {
	uidx := types.Users(users).IndexByID()
	trgdir := channelName(ci)
	lg := logger.FromContext(ctx)
	var (
		prevDt string        // previous date
		enc    *json.Encoder // current encoder
	)
	var wc io.WriteCloser // current file
	defer func() {
		// sentinel to ensure that wc is closed on exit.
		if wc != nil {
			wc.Close()
		}
	}()
	if err := pl.Sorted(ctx, false, func(ts time.Time, m *slack.Message) error {
		date := ts.Format("2006-01-02")
		if date != prevDt || prevDt == "" {
			lg.Debugf("transforming messages for channel: %q, date: %s", ci.ID, date)
			// if we have advanced to the next date, switch to a new file.
			if wc != nil {
				if err := writeJSONFooter(wc); err != nil {
					return fmt.Errorf("error writing JSON footer: %w", err)
				}
				if err := wc.Close(); err != nil {
					return fmt.Errorf("error closing file: %w", err)
				}
			}
			var err error
			wc, err = fsa.Create(filepath.Join(trgdir, date+".json"))
			if err != nil {
				return fmt.Errorf("error creating file in adapter: %w", err)
			}
			if err := writeJSONHeader(wc); err != nil {
				return fmt.Errorf("error writing JSON header: %w", err)
			}
			prevDt = date
			enc = json.NewEncoder(wc)
			enc.SetIndent("", "  ")
		} else {
			_, err := wc.Write([]byte(",\n"))
			if err != nil {
				return fmt.Errorf("error writing JSON separator: %w", err)
			}
		}

		// in original Slack Export, thread starting messages have some thread
		// statistics, and for this we need to scan the chunk file and get it.
		var thread []slack.Message
		if m.ThreadTimestamp == m.Timestamp && m.LatestReply != structures.NoRepliesLatestReply {
			// get the thread for the initial thread message only.
			var err error
			thread, err = pl.AllThreadMessages(ci.ID, m.ThreadTimestamp)
			if err != nil {
				if !errors.Is(err, chunk.ErrNotFound) {
					return fmt.Errorf("error getting thread messages for %q: %w", ci.ID+":"+m.ThreadTimestamp, err)
				} else {
					// this shouldn't happen as we have the guard in the if
					// condition, but if it does (i.e. API changed), log it.
					lg.Printf("not an error, possibly deleted thread: %q not found in chunk file", ci.ID+":"+m.ThreadTimestamp)
				}
			}
		}
		// transform the message
		if err := enc.Encode(toExportMessage(m, thread, uidx[m.User])); err != nil {
			return fmt.Errorf("error encoding message: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("sorted callback error: %w", err)
	}
	// write the last footer
	if wc != nil {
		if err := writeJSONFooter(wc); err != nil {
			return fmt.Errorf("error writing JSON footer (final): %w", err)
		}
		if err := wc.Close(); err != nil {
			return fmt.Errorf("error closing file (final): %w", err)
		}
	}
	return nil
}

// toExportMessage converts a slack message m to an export message, populating
// the fields that are not present in the original message.  To populate the
// count of replies and reply users on a lead message of a thread, it needs
// "thread".  If m is not a parent message (m.ts != m.thread_ts) for the
// thread or thread is nil, it is ignored (this follows the original Slack
// Export logic).  user is the poster's user information.  Original Slack
// Export adds the "profile" field on each message with basic profile
// information about the poster.
func toExportMessage(m *slack.Message, thread []slack.Message, user *slack.User) *export.ExportMessage {
	// export message
	em := export.ExportMessage{
		Msg:        &m.Msg,
		UserTeam:   m.Team,
		SourceTeam: m.Team,
	}

	// add user profile
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

	// add thread information if it is a lead message of a thread.
	if m.Timestamp == m.ThreadTimestamp && len(thread) > 0 {
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
