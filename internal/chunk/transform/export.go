package transform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime/trace"
	"sort"
	"sync/atomic"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/export"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fasttime"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/types"
)

type ExpCvtOption func(*ExpConverter)

func ExpWithMsgUpdateFunc(fn ...func(*slack.Channel, *slack.Message) error) ExpCvtOption {
	return func(t *ExpConverter) {
		for _, f := range fn {
			t.msgFunc = append(t.msgFunc, msgUpdFunc(f))
		}
	}
}

func ExpWithUsers(users []slack.User) ExpCvtOption {
	return func(t *ExpConverter) {
		t.SetUsers(users)
	}
}

type ExpConverter struct {
	cd      *chunk.Directory
	fsa     fsadapter.FS
	users   atomic.Value
	msgFunc []msgUpdFunc
}

func NewExpConverter(cd *chunk.Directory, fsa fsadapter.FS, opt ...ExpCvtOption) *ExpConverter {
	e := &ExpConverter{
		cd:  cd,
		fsa: fsa,
	}
	for _, o := range opt {
		o(e)
	}
	return e
}

func (e *ExpConverter) SetUsers(users []slack.User) {
	e.users.Store(users)
}

func (e *ExpConverter) getUsers() []slack.User {
	uu, ok := e.users.Load().([]slack.User)
	if !ok {
		return nil
	}
	return uu
}

// Convert is the chunk file export converter.  It transforms the chunk file
// for the channel with ID into a slack export format.  It expects the chunk
// file to be in the <srcdir>/<id>.json.gz file, and the attachments to be in
// the <srcdir>/<id> directory.
func (e *ExpConverter) Convert(ctx context.Context, id chunk.FileID) error {
	ctx, task := trace.NewTask(ctx, "transform")
	defer task.End()

	lg := slog.With("file_id", id)
	{
		userCnt := len(e.getUsers())
		trace.Logf(ctx, "input", "len(users)=%d", userCnt)
		lg.DebugContext(ctx, "transforming channel", "id", id, "user_count", userCnt)
	}

	// load the chunk file
	cf, err := e.cd.Open(id)
	if err != nil {
		return fmt.Errorf("error opening chunk file %q: %w", id, err)
	}
	defer cf.Close()

	channelID, _ := id.Split()
	ci, err := cf.ChannelInfo(channelID)
	if err != nil {
		return fmt.Errorf("error reading channel info for %q: %w", id, err)
	}

	if err := e.writeMessages(ctx, cf, ci); err != nil {
		return err
	}

	return nil
}

type filer interface {
	Sorted(context.Context, bool, func(time.Time, *slack.Message) error) error
	AllThreadMessages(channelID, threadTS string) ([]slack.Message, error)
}

func (e *ExpConverter) writeMessages(ctx context.Context, pl filer, ci *slack.Channel) error {
	lg := slog.With("in", "writeMessages", "channel", ci.ID)
	uidx := types.Users(e.getUsers()).IndexByID()
	trgdir := ExportChanName(ci)

	mm := make([]export.ExportMessage, 0, 100)
	var prevDt string
	var currDt string
	if err := pl.Sorted(ctx, false, func(ts time.Time, m *slack.Message) error {
		currDt = ts.Format("2006-01-02")
		if currDt != prevDt || prevDt == "" {
			if prevDt != "" {
				if err := e.writeout(filepath.Join(trgdir, prevDt+".json"), mm); err != nil {
					return err
				}
			}
			mm = make([]export.ExportMessage, 0, 100)
			prevDt = currDt
		}

		// the "thread" is only used to collect statistics.  Thread messages
		// are passed by Sorted and written as a normal course of action.
		var thread []slack.Message
		if structures.IsThreadStart(m) && m.LatestReply != structures.LatestReplyNoReplies {
			// get the thread for the initial thread message only.
			var err error
			thread, err = pl.AllThreadMessages(ci.ID, m.ThreadTimestamp)
			if err != nil {
				if !errors.Is(err, chunk.ErrNotFound) {
					return fmt.Errorf("error getting thread messages for %q: %w", ci.ID+":"+m.ThreadTimestamp, err)
				} else {
					// this shouldn't happen as we have the guard in the if
					// condition, but if it does (i.e. API changed), log it.
					lg.Warn("not an error, possibly deleted thread not found in chunk file", "slack_link", ci.ID+":"+m.ThreadTimestamp)
				}
			}
		}

		// apply all message functions.
		for _, fn := range e.msgFunc {
			if err := fn(ci, m); err != nil {
				return fmt.Errorf("error updating message: %w", err)
			}
		}

		mm = append(mm, *toExportMessage(m, thread, uidx[m.User]))
		return nil
	}); err != nil {
		return fmt.Errorf("sorted callback error: %w", err)
	}

	// flush the last day.
	if len(mm) > 0 {
		if err := e.writeout(filepath.Join(trgdir, prevDt+".json"), mm); err != nil {
			return err
		}
	}

	return nil
}

func (e *ExpConverter) writeout(filename string, mm []export.ExportMessage) error {
	wc, err := e.fsa.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file in adapter: %w", err)
	}
	defer wc.Close()
	enc := json.NewEncoder(wc)
	enc.SetIndent("", "  ")
	if err := enc.Encode(mm); err != nil {
		return fmt.Errorf("error encoding messages: %w", err)
	}
	return nil
}

type msgUpdFunc func(*slack.Channel, *slack.Message) error

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
		UserTeam:   m.Team, // TODO: user is lacking team, so using the message team.
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
		sort.Slice(em.Replies, func(i, j int) bool {
			tsi, err := fasttime.TS2int(em.Msg.Replies[i].Timestamp)
			if err != nil {
				return false
			}
			tsj, err := fasttime.TS2int(em.Msg.Replies[j].Timestamp)
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

// TODO: replace with an stdlib function.
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

// ExportChanName returns the channel name, or the channel ID if it is a DM.
func ExportChanName(ch *slack.Channel) string {
	if ch.IsIM {
		return ch.ID
	}
	return ch.Name
}

// WriteIndex generates and writes the export index files.  It must be called
// once all transformations are done, because it might require to read channel
// files.
func (t *ExpConverter) WriteIndex(ctx context.Context) error {
	wsp, err := t.cd.WorkspaceInfo()
	if err != nil {
		return fmt.Errorf("failed to get the workspace info: %w", err)
	}
	chans, err := t.cd.Channels(ctx)
	if err != nil {
		return fmt.Errorf("error indexing channels: %w", err)
	}
	eidx, err := structures.MakeExportIndex(chans, t.getUsers(), wsp.UserID)
	if err != nil {
		return fmt.Errorf("error creating export index: %w", err)
	}
	if err := eidx.Marshal(t.fsa); err != nil {
		return fmt.Errorf("error writing export index: %w", err)
	}
	return nil
}

// HasUsers returns true if the converter has users.
func (t *ExpConverter) HasUsers() bool {
	return len(t.getUsers()) > 0
}
