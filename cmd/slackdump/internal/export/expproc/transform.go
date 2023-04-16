package expproc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/trace"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rusq/dlog"
	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/osext"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
)

type Transform struct {
	srcdir  string       // source directory with chunks.
	fsa     fsadapter.FS // target file system adapter.
	ids     chan string  // channel used to pass channel IDs to the worker.
	done    chan struct{}
	users   []slack.User // list of users.
	err     chan error   // error channel used to propagate errors to the main thread.
	started atomic.Bool
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

// NewTransform creates a new Transform instance.  The fsa is the filesystem
// adapter that holds the transformed data (output), chunkdir is the directory
// where the chunks, produced by processor, are stored.
func NewTransform(ctx context.Context, fsa fsadapter.FS, chunkdir string, tfopt ...TfOption) (*Transform, error) {
	if err := osext.DirExists(chunkdir); err != nil {
		return nil, fmt.Errorf("chunk directory %s does not exist: %w", chunkdir, err)
	}
	t := &Transform{
		srcdir: chunkdir,
		fsa:    fsa,
	}
	for _, opt := range tfopt {
		opt(t)
	}
	return t, nil
}

// WriteUsers writes the list of users to the file system adapter.
func (t *Transform) WriteUsers(users []slack.User) error {
	if users == nil {
		return errors.New("users list is nil")
	}
	return t.writeUsers(users)
}

// Start starts the Transform processor with the provided list of users.
// Users are used to populate each message with the user profile, as per Slack
// original export format.
func (t *Transform) StartWithUsers(ctx context.Context, users []slack.User) error {
	if users == nil {
		return errors.New("users list is nil")
	}
	t.users = users
	return t.Start(ctx)
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

// Start starts the Transform processor, the users must have been initialised
// with the WithUsers option.  Otherwise, use StartWithUsers method.
func (t *Transform) Start(ctx context.Context) error {
	if t.users == nil {
		return errors.New("internal error: users not initialised")
	}
	t.ids = make(chan string)
	t.done = make(chan struct{})
	t.err = make(chan error, 1)

	t.started.Store(false)
	go t.worker(ctx)
	return nil
}

// OnFinalise is the function that should be passed to the Channel processor.
// It will not block if the internal buffer is full.  Buffer size can be
// set with the WithBufferSize option.
func (t *Transform) OnFinalise(channelID string) error {
	if !t.started.Load() {
		return errors.New("transformer not started")
	}
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
	if !t.started.Load() {
		return nil
	}
	t.Stop()
	return nil
}

// Stop stops the Transform processor and waits for all workers to finish what
// they're doing.  It can be called multiple times, if the processor is
// already stopped, it will return nil.
//
// Stop MUST be called before writing the user list and other things for it
// not to interfere with the worker.
func (t *Transform) Stop() {
	if !t.started.Load() {
		return
	}
	close(t.ids)
	<-t.done
	t.started.Store(false)
}

// transform is the chunk file transformer.  It transforms the chunk file for
// the channel with ID into a slack export format, and attachments are placed
// into the relevant directory.  It expects the chunk file to be in the
// srcdir/id.json.gz file, and the attachments to be in the srcdir/id
// directory.
func transform(ctx context.Context, fsa fsadapter.FS, srcdir string, id string, users []slack.User) error {
	ctx, task := trace.NewTask(ctx, "transform")
	defer task.End()
	lg := dlog.FromContext(ctx)
	trace.Logf(ctx, "input", "len(users)=%d", len(users))
	lg.Debugf("transforming channel %s, user len=%d", id, len(users))

	// load the chunk file
	f, err := openChunks(filepath.Join(srcdir, id+ext))
	if err != nil {
		return err
	}
	defer f.Close()
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

// LoadUsers loads the list of users from the chunk file.
func LoadUsers(ctx context.Context, dir string) ([]slack.User, error) {
	_, task := trace.NewTask(ctx, "load users")
	defer task.End()

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
	trgdir := channelName(ci)
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
			wc, err = fsa.Create(filepath.Join(trgdir, date+".json"))
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

		// in original Slack Export, thread starting messages have some thread
		// statistics, and for this we need to scan the chunk file and get it.
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
		return enc.Encode(toExportMessage(m, thread, uidx[m.User]))
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

// toExportMessage converts a slack message to an export message.
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

// openChunks opens a chunk file and returns a ReadSeekCloser.  It expects
// a chunkfile to be a gzip-compressed file.
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

	return osext.RemoveOnClose(tf), nil
}

var errNoChannelInfo = errors.New("no channel info")

// LoadChannels collects all channels from the chunk directory.  First,
// it attempts to find the channel.json.gz file, if it's not present, it will
// go through all conversation files and try to get "ChannelInfo" chunk from
// the each file.
func LoadChannels(dir string) ([]slack.Channel, error) {
	// try to open the channels file
	const channelsJSON = "channels" + ext
	if fi, err := os.Stat(filepath.Join(dir, channelsJSON)); err == nil && !fi.IsDir() {
		return loadChannelsJSON(filepath.Join(dir, channelsJSON))
	}
	// channel files not found, try to get channel info from the conversation
	// files.
	var ch []slack.Channel
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ext) {
			return nil
		} else if d.IsDir() {
			return nil
		}
		chs, err := loadChanInfo(path)
		if err != nil {
			if errors.Is(err, errNoChannelInfo) {
				return nil
			}
			return err
		}
		ch = append(ch, chs...)
		return nil
	}); err != nil {
		return nil, err
	}
	return ch, nil
}

func loadChanInfo(fullpath string) ([]slack.Channel, error) {
	f, err := openChunks(fullpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readChanInfo(f)
}

func readChanInfo(rs io.ReadSeeker) ([]slack.Channel, error) {
	cf, err := chunk.FromReader(rs)
	if err != nil {
		return nil, err
	}
	return cf.AllChannelInfos()
}

// loadChannelsJSON loads channels json file and returns a slice of
// slack.Channel.  It expects it to be GZIP compressed.
func loadChannelsJSON(fullpath string) ([]slack.Channel, error) {
	cf, err := openChunks(fullpath)
	if err != nil {
		return nil, err
	}
	defer cf.Close()
	return readChannelsJSON(cf)
}

func readChannelsJSON(r io.Reader) ([]slack.Channel, error) {
	var ch []slack.Channel
	if err := json.NewDecoder(r).Decode(&ch); err != nil {
		return nil, err
	}
	return ch, nil
}
