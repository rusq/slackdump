package list

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
	"github.com/rusq/slackdump/v3/internal/format"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/rusq/slackdump/v3/types"
)

const (
	userCacheBase = "users.cache"
)

// CmdList is the list command.  The logic is in the subcommands.
var CmdList = &base.Command{
	UsageLine: "slackdump list",
	Short:     "list users or channels",
	Long: fmt.Sprintf(`
# List Command

List lists users or channels for the Slack Workspace.  It may take a while on a
large workspace, as Slack limits the amount of requests on it's own discretion,
which is sometimes unreasonably slow.

The data is dumped to a JSON file in the base directory, and additionally,
printed on the screen in the requested format.

- To disable saving data to a file, use '-no-save' flag.
- To disable printing on the screen, use '-q' (quiet) flag.

## Caching
Channel and User data is cached.  Default user cache retention is %s, and
channel cache — %s.  This is to speed up consecutive runs of the command.

The caching can be turned off by using flags "-no-user-cache" and
"-no-chan-cache".
`, cfg.UserCacheRetention, chanCacheOpts.Retention),
	Commands: []*base.Command{
		CmdListUsers,
		CmdListChannels,
	},
}

// common flags
var (
	listType format.Type = format.CText
	quiet    bool        // quiet mode:  don't print anything on the screen, just save the file
	nosave   bool        // nosave mode:  don't save the data to a file, just print it to the screen
)

func init() {
	for _, cmd := range CmdList.Commands {
		addCommonFlags(&cmd.Flag)
	}
}

// addCommonFlags adds common flags to the flagset.
func addCommonFlags(fs *flag.FlagSet) {
	fs.Var(&listType, "format", fmt.Sprintf("listing format, should be one of: %v", format.All()))
	fs.BoolVar(&quiet, "q", false, "quiet mode:  don't print anything on the screen, just save the file")
	fs.BoolVar(&nosave, "no-json", false, "don't save the data to a file, just print it to the screen")
}

// listFunc is a function that lists something from the Slack API.  It should
// return the object from the api, a filename to save the data to and an
// error.
type listFunc func(ctx context.Context, sess *slackdump.Session) (a any, filename string, err error)

// list authenticates and creates a slackdump instance, then calls a listFn.
// listFn must return the object from the api, a JSON filename and an error.
func list(ctx context.Context, listFn listFunc) error {
	// TODO fix users saving JSON to a text file within archive
	if listType == format.CUnknown {
		return errors.New("unknown listing format, seek help")
	}

	// initialize the session.
	sess, err := cfg.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	data, filename, err := listFn(ctx, sess)
	if err != nil {
		return err
	}
	m, err := cache.NewManager(cfg.CacheDir(), cache.WithUserCacheBase(userCacheBase))
	if err != nil {
		return err
	}

	teamID := sess.Info().TeamID
	users, ok := data.(types.Users)
	if !ok {
		users, err = getCachedUsers(ctx, sess, m, teamID)
		if err != nil {
			return err
		}
	}

	if !nosave {
		fsa, err := fsadapter.New(cfg.Output)
		if err != nil {
			return err
		}
		defer fsa.Close()
		if err := saveData(ctx, fsa, data, filename, format.CJSON, users); err != nil {
			return err
		}
	}

	if !quiet {
		return fmtPrint(ctx, os.Stdout, data, listType, users)
	}

	return nil
}

// saveData saves the given data to the given filename.
func saveData(ctx context.Context, fs fsadapter.FS, data any, filename string, typ format.Type, users []slack.User) error {
	// save to a filesystem.
	f, err := fs.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	if err := fmtPrint(ctx, f, data, typ, users); err != nil {
		return err
	}
	logger.FromContext(ctx).Printf("Data saved to:  %q\n", filepath.Join(cfg.Output, filename))

	return nil
}

type userGetter interface {
	GetUsers(ctx context.Context) (types.Users, error)
}

type userCacher interface {
	LoadUsers(teamID string, retention time.Duration) ([]slack.User, error)
	CacheUsers(teamID string, users []slack.User) error
}

func getCachedUsers(ctx context.Context, ug userGetter, m userCacher, teamID string) ([]slack.User, error) {
	lg := logger.FromContext(ctx)

	users, err := m.LoadUsers(teamID, cfg.UserCacheRetention)
	if err == nil {
		return users, nil
	}

	// failed to load from cache
	if !errors.Is(err, cache.ErrExpired) && !errors.Is(err, cache.ErrEmpty) {
		// some funky error
		return nil, err
	}

	lg.Println("user cache expired or empty, caching users")

	// getting users from API
	users, err = ug.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	// saving users to cache, will ignore any errors, but notify the user.
	if err := m.CacheUsers(teamID, users); err != nil {
		trace.Logf(ctx, "error", "saving user cache to %q, error: %s", userCacheBase, err)
		lg.Printf("warning: failed saving user cache to %q: %s, but nevermind, let's continue", userCacheBase, err)
	}

	return users, nil
}

// fmtPrint prints the given data to the given writer, using the given format.
// It should be supplied with prepopulated users, as it may need to look up
// users by ID.
func fmtPrint(ctx context.Context, w io.Writer, a any, typ format.Type, u []slack.User) error {
	// get the converter
	initFn, ok := format.Converters[typ]
	if !ok {
		return fmt.Errorf("unknown converter type: %s", typ)
	}
	cvt := initFn()

	// currently there's no list function for conversations, because it
	// requires additional options, and I don't want to clutter the flags -
	// there's already too many.
	switch val := a.(type) {
	case types.Channels:
		return cvt.Channels(ctx, w, u, val)
	case types.Users:
		return cvt.Users(ctx, w, val)
	case *types.Conversation:
		return cvt.Conversation(ctx, w, u, val)
	default:
		return fmt.Errorf("unsupported data type: %T", a)
	}
	// unreachable
}

// makeFilename makes a filename for the given prefix, teamID and listType for
// channels and users.
func makeFilename(prefix string, teamID string, ext string) string {
	return fmt.Sprintf("%s-%s%s", prefix, teamID, ext)
}

func wizard(ctx context.Context, listFn listFunc) error {
	// pick format
	var types []string
	for _, t := range format.All() {
		types = append(types, t.String())
	}

	var listType format.Type
	var ot string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[format.Type]().Title("Pick a format").Options(huh.NewOptions(format.All()...)...).Value(&listType),
			huh.NewSelect[string]().Title("Pick an output type").Options(huh.NewOptions("screen", "ZIP file", "directory")...).Value(&ot),
		))
	if err := form.Run(); err != nil {
		return err
	}
	if ot != "screen" {
		return errors.New("not implemented yet")
	}
	// if file/directory, pick filename
	return nil
}
