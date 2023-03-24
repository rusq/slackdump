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

	"github.com/AlecAivazis/survey/v2"
	"github.com/rusq/dlog"
	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/cache"
	"github.com/rusq/slackdump/v2/internal/format"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
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

	// get the provider from Context.
	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}

	// initialize the session.
	sess, err := slackdump.New(ctx, prov, slackdump.WithLogger(dlog.FromContext(ctx)))
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
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

	lg := dlog.FromContext(ctx)
	teamID := sess.Info().TeamID
	users, err := m.LoadUsers(teamID, cfg.UserCacheRetention)
	if err != nil {
		lg.Println("user cache expired, caching users")

		users, err = sess.GetUsers(ctx)
		if err != nil {
			return err
		}

		if err := m.SaveUsers(teamID, users); err != nil {
			trace.Logf(ctx, "error", "saving user cache to %q, error: %s", userCacheBase, err)
			lg.Printf("error saving user cache to %q: %s, but nevermind, let's continue", userCacheBase, err)
		}
	}

	if !nosave {
		fsa, err := fsadapter.New(cfg.BaseLocation)
		if err != nil {
			return err
		}
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
	if err := writeData(ctx, fs, filename, data, typ, users); err != nil {
		return err
	}
	dlog.FromContext(ctx).Printf("Data saved to:  %q\n", filepath.Join(cfg.BaseLocation, filename))
	return nil
}

func writeData(ctx context.Context, fs fsadapter.FS, filename string, data any, typ format.Type, users []slack.User) error {
	f, err := fs.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	return fmtPrint(ctx, f, data, typ, users)
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

// extmap maps a format.Type to a file extension.
var extmap = map[format.Type]string{
	format.CText: "txt",
	format.CJSON: "json",
	format.CCSV:  "csv",
}

// makeFilename makes a filename for the given prefix, teamID and listType for
// channels and users.
func makeFilename(prefix string, teamID string, listType format.Type) string {
	ext, ok := extmap[listType]
	if !ok {
		panic(fmt.Sprintf("unknown list type: %v", listType))
	}
	return fmt.Sprintf("%s-%s.%s", prefix, teamID, ext)
}

func wizard(ctx context.Context, listFn listFunc) error {
	// pick format
	var types []string
	for _, t := range format.All() {
		types = append(types, t.String())
	}

	q := &survey.Select{
		Message: "Pick a format:",
		Options: types,
		Help:    "Pick a format for the listing",
		Description: func(value string, index int) string {
			var v format.Type
			v.Set(value)
			return format.Descriptions[v]
		},
	}
	var lt int
	if err := survey.AskOne(q, &lt); err != nil {
		return err
	}
	listType = format.Type(lt)
	// pick output type: screen or file/directory
	q = &survey.Select{
		Message: "Pick an output type:",
		Options: []string{"screen", "ZIP file", "directory"},
		Help:    "Pick an output type for the listing",
	}
	var ot string
	if err := survey.AskOne(q, &ot); err != nil {
		return err
	}
	if ot != "screen" {
		return errors.New("not implemented yet")
	}

	// if file/directory, pick filename
	return nil
}
