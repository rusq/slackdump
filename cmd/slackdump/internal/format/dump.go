package format

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"runtime/trace"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/cache"
	"github.com/rusq/slackdump/v3/internal/format"
	"github.com/rusq/slackdump/v3/types"
)

// formatJSONfile formats a single json file in the dump format.
func formatJSONfile(ctx context.Context, w io.Writer, cvt format.Formatter, rs io.ReadSeeker) error {
	ctx, task := trace.NewTask(ctx, "convert")
	defer task.End()
	lg := cfg.Log

	dump, err := detectAndRead(rs)
	if err != nil {
		return ErrInvalidFormat
	}
	lg.InfoContext(ctx, "Successfully detected file type", "type", dump.filetype)

	if dump.filetype == dtUsers {
		// special case.  Users do not need to have users fetched from slack etc,
		// because, well, because they are users already.
		return cvt.Users(ctx, w, dump.users)
	}

	uu, err := getUsers(ctx, dump, flgOnline)
	if err != nil {
		return err
	}

	switch dump.filetype {
	case dtChannels:
		return cvt.Channels(ctx, w, uu, dump.channels)
	case dtConversation:
		return cvt.Conversation(ctx, w, uu, dump.conversation)
	case dtUnknown:
		fallthrough
	default:
	}
	return errors.New("internal error: undetected type")
}

//go:generate stringer -type dumptype -trimprefix=dt
type dumptype uint8

const (
	dtUnknown dumptype = iota
	dtConversation
	dtChannels
	dtUsers
)

var ErrUnknown = errors.New("unknown file type")

type idextractor interface {
	UserIDs() []string
}

var ErrInvalidFormat = errors.New("not a dump JSON file")

// dump represents a slack data dump.  Only one variable will be initialised
// depending on the dumptype.
type dump struct {
	filetype     dumptype
	users        types.Users
	channels     types.Channels
	conversation *types.Conversation
}

// detectAndRead detects the filetype by consequently trying to unmarshal the
// data.  It will return [dump] that will have [dumptype] and one of the
// member variables populated.  If it fails to detect the type it will return
// ErrUnknown and set the dump filetype to dtUnknown.
func detectAndRead(rs io.ReadSeeker) (*dump, error) {
	d := new(dump)

	if conv, err := unmarshal[types.Conversation](rs); err != nil && !isJSONTypeErr(err) {
		return nil, err
	} else if conv.ID != "" {
		d.filetype = dtConversation
		d.conversation = &conv
		return d, nil
	}

	if ch, err := unmarshal[[]slack.Channel](rs); err != nil && !isJSONTypeErr(err) {
		return nil, err
	} else if len(ch) > 0 && ch[0].Creator != "" {
		d.filetype = dtChannels
		d.channels = ch
		return d, nil
	}

	if u, err := unmarshal[[]slack.User](rs); err != nil && !isJSONTypeErr(err) {
		return nil, err
	} else if len(u) > 0 && u[0].RealName != "" {
		d.filetype = dtUsers
		d.users = u
		return d, nil
	}

	// no luck
	d.filetype = dtUnknown
	return d, ErrUnknown
}

func isJSONTypeErr(err error) bool {
	var e *json.UnmarshalTypeError
	return errors.As(err, &e)
}

func (d dump) userIDs() []string {
	var xt idextractor
	switch d.filetype {
	case dtConversation:
		xt = d.conversation
	case dtUsers:
		xt = d.users
	case dtChannels:
		xt = d.channels
	}
	return xt.UserIDs()
}

func unmarshal[OUT any](rs io.ReadSeeker) (OUT, error) {
	var ret OUT
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return ret, err
	}
	defer rs.Seek(0, io.SeekStart)

	dec := json.NewDecoder(rs)
	if err := dec.Decode(&ret); err != nil {
		return ret, err
	}

	return ret, nil
}

func getUsers(ctx context.Context, dmp *dump, isOnline bool) ([]slack.User, error) {
	if isOnline {
		return getUsersOnline(ctx)
	}
	rgn := trace.StartRegion(ctx, "userIDs")
	ids := dmp.userIDs()
	rgn.End()
	if len(ids) == 0 {
		return nil, errors.New("unable to extract user IDs")
	}
	trace.Logf(ctx, "getUsers", "number of users in this dump: %d", len(ids))
	uu, err := searchCache(ctx, cfg.CacheDir(), ids)
	if err != nil {
		return nil, err
	}
	return uu, nil
}

func getUsersOnline(ctx context.Context) ([]slack.User, error) {
	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		return nil, err
	}
	return sess.GetUsers(ctx)
}

var errNoMatch = errors.New("no matching users")

// searchCache searches the cache directory for cached workspace users that have
// the same ids, and returns the user slice from that cache.
func searchCache(ctx context.Context, cacheDir string, ids []string) ([]slack.User, error) {
	_, task := trace.NewTask(ctx, "searchCache")
	defer task.End()
	m, err := cache.NewManager(cacheDir, cache.WithMachineID(cfg.MachineIDOvr), cache.WithNoEncryption(cfg.NoEncryption))
	if err != nil {
		return nil, err
	}
	var users []slack.User
	err = m.WalkUsers(func(path string, r io.Reader) error {
		var err1 error
		users, err1 = matchUsers(r, ids)
		if err1 != nil {
			if errors.Is(err1, errNoMatch) {
				return nil
			}
			return err1
		}
		slog.InfoContext(ctx, "matching file", "path", path)
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}
	return users, nil
}

func matchUsers(r io.Reader, ids []string) ([]slack.User, error) {
	const matchRatio = 0.5 // 50% of users must match.
	uu, err := cache.ReadUsers(r)
	if err != nil {
		return nil, err
	}
	fileIDs := uu.IndexByID()
	matchingCnt := 0 // matching users count
	for _, id := range ids {
		if fileIDs[id] != nil {
			matchingCnt++
		}
	}
	if float64(matchingCnt)/float64(len(ids)) < matchRatio {
		return nil, errNoMatch
	}
	return uu, nil
}
