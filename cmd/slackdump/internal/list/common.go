package list

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace"
	"github.com/rusq/slackdump/v3/internal/cache"
	"github.com/rusq/slackdump/v3/internal/format"
	"github.com/rusq/slackdump/v3/types"
)

const flagMask = cfg.OmitDownloadFlag | cfg.OmitMemberOnlyFlag

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
`, cfg.UserCacheRetention, chanFlags.cache.Retention),
	Commands: []*base.Command{
		CmdListUsers,
		CmdListChannels,
	},
}

type lister[T any] interface {
	// Type should return the type of the lister.
	Type() string
	// Retrieve should retrieve the data from the API or cache.
	Retrieve(ctx context.Context, sess *slackdump.Session, m *cache.Manager) error
	// Data should return the retrieved data.
	Data() T
	// Users should return the users for the data, or nil, which indicates
	// that there are no associated users or that the users are not resolved.
	Users() []slack.User
}

// common flags
type commonOpts struct {
	listType format.Type
	quiet    bool // quiet mode:  don't print anything on the screen, just save the file
	nosave   bool // nosave mode:  don't save the data to a file, just print it to the screen
}

var commonFlags = commonOpts{
	listType: format.CText,
}

func init() {
	for _, cmd := range CmdList.Commands {
		addCommonFlags(&cmd.Flag)
	}
}

// addCommonFlags adds common flags to the flagset.
func addCommonFlags(fs *flag.FlagSet) {
	fs.Var(&commonFlags.listType, "format", fmt.Sprintf("listing format, should be one of: %v", format.All()))
	fs.BoolVar(&commonFlags.quiet, "q", false, "quiet mode:  don't print anything on the screen, just save the file")
	fs.BoolVar(&commonFlags.nosave, "no-json", false, "don't save the data to a file, just print it to the screen")
}

func list[T any](ctx context.Context, sess *slackdump.Session, l lister[T], filename string) error {
	m, err := workspace.CacheMgr()
	if err != nil {
		return err
	}

	if err := l.Retrieve(ctx, sess, m); err != nil {
		return err
	}

	if !commonFlags.quiet {
		if err := fmtPrint(ctx, os.Stdout, l.Data(), commonFlags.listType, l.Users()); err != nil {
			return err
		}
	}

	if !commonFlags.nosave {
		if filename == "" {
			filename = makeFilename(l.Type(), sess.Info().TeamID, extForType(commonFlags.listType))
		}
		if err := saveData(ctx, l.Data(), filename, commonFlags.listType, l.Users()); err != nil {
			return err
		}
	}
	return nil
}

func extForType(typ format.Type) string {
	switch typ {
	case format.CJSON:
		return ".json"
	case format.CText:
		return ".txt"
	case format.CCSV:
		return ".csv"
	default:
		return ".json"
	}
}

// saveData saves the given data to the given filename.
func saveData(ctx context.Context, data any, filename string, typ format.Type, users []slack.User) error {
	// save to a filesystem.
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	if err := fmtPrint(ctx, f, data, typ, users); err != nil {
		return err
	}
	cfg.Log.InfoContext(ctx, "Data saved", "filename", filename)

	return nil
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
