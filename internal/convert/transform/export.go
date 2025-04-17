package transform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime/trace"
	"sync/atomic"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/export"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/source"
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
	src     source.Sourcer
	fsa     fsadapter.FS
	users   atomic.Value
	msgFunc []msgUpdFunc
}

func NewExpConverter(src source.Sourcer, fsa fsadapter.FS, opt ...ExpCvtOption) *ExpConverter {
	e := &ExpConverter{
		src: src,
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

	channelID, _ := id.Split()
	ci, err := e.src.ChannelInfo(ctx, channelID)
	if err != nil {
		return fmt.Errorf("error reading channel info for %q: %w", id, err)
	}

	if err := e.writeMessages(ctx, ci); err != nil {
		return fmt.Errorf("error writing messages for %q: %w", id, err)
	}

	return nil
}

func (e *ExpConverter) writeMessages(ctx context.Context, ci *slack.Channel) (err error) {
	lg := slog.With("in", "writeMessages", "channel", ci.ID)
	acc := e.newAccumulator(ctx, ci)
	defer func() {
		e := acc.Flush()
		err = errors.Join(err, e)
	}()
	if err := e.src.Sorted(ctx, ci.ID, false, acc.Append); err != nil {
		if errors.Is(err, source.ErrNotFound) {
			lg.DebugContext(ctx, "no messages for the channel", "channel", ci.ID)
			return nil
		}
		return fmt.Errorf("sorted: on channel %q: %w", ci.ID, err)
	}

	return nil
}

func (e *ExpConverter) writeout(filename string, mm []export.ExportMessage) error {
	wc, err := e.fsa.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file in adapter: %s:  %w", filename, err)
	}
	defer wc.Close()
	enc := json.NewEncoder(wc)
	enc.SetIndent("", "  ")
	if err := enc.Encode(mm); err != nil {
		return fmt.Errorf("error encoding messages into %s: %w", filename, err)
	}
	return nil
}

type msgUpdFunc func(*slack.Channel, *slack.Message) error

// toExportMessage converts a slack message m to an export message, populating
// the fields that are not present in the original message.  To populate the
// count of replies and reply users on a lead message of a thread, it needs
// "thread".  If m is not a parent message (m.ts != m.thread_ts) for the thread
// or thread is nil, it is ignored (this follows the original Slack Export
// logic).  user is the poster's user information.  Original Slack Export adds
// the "profile" field on each message with basic profile information about the
// poster.
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
	if structures.IsThreadStart(m) && len(thread) > 0 {
		em.PopulateReplyFields(thread)
	}

	return &em
}

// WriteIndex generates and writes the export index files.  It must be called
// once all transformations are done, because it might require to read channel
// files.
func (e *ExpConverter) WriteIndex(ctx context.Context) error {
	wsp, err := e.src.WorkspaceInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the workspace info: %w", err)
	}
	chans, err := e.src.Channels(ctx)
	if err != nil {
		return fmt.Errorf("error indexing channels: %w", err)
	}
	eidx, err := structures.MakeExportIndex(chans, e.getUsers(), wsp.UserID)
	if err != nil {
		return fmt.Errorf("error creating export index: %w", err)
	}
	if err := eidx.Marshal(e.fsa); err != nil {
		return fmt.Errorf("error writing export index: %w", err)
	}
	return nil
}

// HasUsers returns true if the converter has users.
func (e *ExpConverter) HasUsers() bool {
	return len(e.getUsers()) > 0
}

func (e *ExpConverter) newAccumulator(ctx context.Context, channel *slack.Channel) *expmsgAccum {
	return &expmsgAccum{
		ctx:     ctx,
		channel: channel,
		src:     e.src,
		trgdir:  source.ExportChanName(channel),
		uidx:    types.Users(e.getUsers()).IndexByID(),
		msgfunc: e.msgFunc,
		flushFn: e.writeout,
	}
}

// expmsgAccum is the message accumulator for the export conversion.
type expmsgAccum struct {
	mm             []export.ExportMessage
	prevDt, currDt string

	src    source.Sourcer
	trgdir string

	ctx     context.Context
	channel *slack.Channel
	msgfunc []msgUpdFunc
	uidx    map[string]*slack.User
	flushFn func(filename string, mm []export.ExportMessage) error
}

const (
	msgBufSz    = 50 // on average, on a low-load medium size workspace.
	threadBufSz = 5  // average thread size is 1.4 messages.
)

func (a *expmsgAccum) next() {
	a.mm = make([]export.ExportMessage, 0, msgBufSz)
	a.prevDt = a.currDt
}

func (a *expmsgAccum) shouldFlush() bool {
	return a.currDt != a.prevDt || a.prevDt == ""
}

// Seattle timezone
var exportLoc, _ = time.LoadLocation("America/Los_Angeles")

// Append appends a message to the accumulator.  It flushes the messages to the
// file when the date changes. It also updates the message with the user
// profile information and thread information if it is a lead message of a
// thread.
func (a *expmsgAccum) Append(ts time.Time, m *slack.Message) error {
	a.currDt = ts.In(exportLoc).Format("2006-01-02")
	if a.shouldFlush() {
		// flush the previous day.
		if err := a.Flush(); err != nil {
			return err
		}
		a.next()
	}

	// NOTE:  the "thread" is only used to collect statistics.  Thread messages
	// are passed by Sorted along with channel messages.
	var thread []slack.Message
	if structures.IsThreadStart(m) && !structures.IsEmptyThread(m) {
		// get the thread for the initial thread message only.
		itTm, err := a.src.AllThreadMessages(a.ctx, a.channel.ID, m.ThreadTimestamp)
		if err != nil {
			if errors.Is(err, source.ErrNotFound) || errors.Is(err, source.ErrNotSupported) {
				slog.Warn("not an error, possibly deleted thread not found in chunk file", "slack_link", a.channel.ID+":"+m.ThreadTimestamp, "in", "Append", "channel", a.channel.ID)
				return nil
			}
			return err
		}
		thread = make([]slack.Message, 0, threadBufSz)
		for tm, err := range itTm {
			if err != nil {
				return fmt.Errorf("error reading thread message: %w", err)
			}
			thread = append(thread, tm)
		}
	}

	// apply all message functions.
	for _, fn := range a.msgfunc {
		if err := fn(a.channel, m); err != nil {
			return fmt.Errorf("error updating message: %w", err)
		}
	}

	a.mm = append(a.mm, *toExportMessage(m, thread, a.uidx[m.User]))
	return nil
}

func (a *expmsgAccum) Flush() error {
	if a.prevDt != "" && len(a.mm) > 0 {
		return a.flushFn(filepath.Join(a.trgdir, a.prevDt+".json"), a.mm)
	}
	return nil
}
